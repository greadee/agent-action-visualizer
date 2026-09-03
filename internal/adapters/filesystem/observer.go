// Package filesystem implements the generic wrapper's metadata-only,
// failure-open filesystem fallback.
package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	"github.com/greadee/agent-action-visualizer/adapter/go/wrapper"
	"github.com/greadee/agent-action-visualizer/internal/project"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

const (
	DefaultQueueSize    = 1024
	DefaultMaxPending   = 512
	DefaultBatchSize    = 128
	DefaultDebounce     = 75 * time.Millisecond
	DefaultRenameWindow = 150 * time.Millisecond
)

type Config struct {
	QueueSize    int
	MaxPending   int
	BatchSize    int
	Debounce     time.Duration
	RenameWindow time.Duration
	Git          GitInspector
	Now          func() time.Time

	newSource func(int) (eventSource, error)
	onReady   func()
}

type Observer struct {
	config Config
}

type fileState struct {
	exists    bool
	directory bool
	size      int64
	modTime   int64
}

type pendingChange struct {
	path   string
	op     operation
	before fileState
	after  fileState
	first  time.Time
	last   time.Time
	count  int
}

type queuedChange struct {
	change change
	at     time.Time
}

func NewObserver(config Config) *Observer {
	if config.QueueSize <= 0 {
		config.QueueSize = DefaultQueueSize
	}
	if config.MaxPending <= 0 {
		config.MaxPending = DefaultMaxPending
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.Debounce <= 0 {
		config.Debounce = DefaultDebounce
	}
	if config.RenameWindow <= 0 {
		config.RenameWindow = DefaultRenameWindow
	}
	if config.Git == nil {
		config.Git = NewCommandGitInspector(250 * time.Millisecond)
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.newSource == nil {
		config.newSource = newFSNotifySource
	}
	return &Observer{config: config}
}

func (o *Observer) Observe(ctx context.Context, observation wrapper.Observation, emitter adapter.Emitter) {
	defer func() { _ = recover() }()
	if o == nil || emitter == nil {
		return
	}
	root, err := filepath.Abs(observation.ProjectRoot)
	if err != nil {
		return
	}
	source, err := o.config.newSource(o.config.QueueSize)
	if err != nil {
		return
	}
	defer source.Close()

	matcher := project.NewIgnoreMatcher(root)
	states, err := watchTree(ctx, source, root, root, matcher)
	if err != nil {
		return
	}
	raw := make(chan queuedChange, o.config.QueueSize)
	var dropped atomic.Uint64
	drainReady := make(chan struct{})
	go drainSource(ctx, source, raw, &dropped, o.config.Now, drainReady)
	<-drainReady
	if o.config.onReady != nil {
		o.config.onReady()
	}

	pending := make(map[string]*pendingChange)
	tickEvery := max(10*time.Millisecond, o.config.Debounce/2)
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	sequence := uint64(0)

	flush := func(flushAll bool) {
		ready := readyChanges(pending, o.config.Now(), o.config.Debounce, o.config.RenameWindow, flushAll, o.config.BatchSize)
		if len(ready) == 0 && dropped.Load() == 0 {
			return
		}
		events := o.eventsForBatch(ctx, observation, root, ready, &sequence)
		if lost := dropped.Swap(0); lost > 0 {
			sequence++
			events = append(events, droppedEvent(observation, sequence, o.config.Now(), lost))
		}
		emitter.Emit(context.Background(), events)
	}

	for {
		select {
		case <-ctx.Done():
			flush(true)
			return
		case item, ok := <-raw:
			if !ok {
				flush(true)
				return
			}
			o.acceptChange(ctx, source, root, matcher, states, pending, item, &dropped)
			if isIgnoreFile(item.change.path) {
				matcher.Reload()
			}
		case <-ticker.C:
			flush(false)
		}
	}
}

func drainSource(ctx context.Context, source eventSource, output chan<- queuedChange, dropped *atomic.Uint64, now func() time.Time, ready chan<- struct{}) {
	defer close(output)
	close(ready)
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-source.Errors():
			if !ok {
				return
			}
			dropped.Add(1)
		case event, ok := <-source.Events():
			if !ok {
				return
			}
			item := queuedChange{change: event, at: now().UTC()}
			select {
			case output <- item:
			default:
				dropped.Add(1)
			}
		}
	}
}

func watchTree(ctx context.Context, source eventSource, root, start string, matcher *project.IgnoreMatcher) (map[string]fileState, error) {
	states := make(map[string]fileState)
	err := filepath.WalkDir(start, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if path != root {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if matcher.Ignored(relative, entry.IsDir()) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		clean := filepath.Clean(path)
		states[clean] = stateFromInfo(info)
		if entry.IsDir() {
			if err := source.Add(clean); err != nil {
				return err
			}
		}
		return nil
	})
	return states, err
}

func (o *Observer) acceptChange(
	ctx context.Context,
	source eventSource,
	root string,
	matcher *project.IgnoreMatcher,
	states map[string]fileState,
	pending map[string]*pendingChange,
	item queuedChange,
	dropped *atomic.Uint64,
) {
	path, allowed := observedPath(root, item.change.path, matcher)
	if !allowed {
		return
	}
	before := states[path]
	after := statState(path)
	if after.exists && isSymlink(path) {
		return
	}
	if item.change.op&opCreate != 0 && after.directory {
		added, err := watchTree(ctx, source, root, path, matcher)
		if err == nil {
			for child, state := range added {
				states[child] = state
			}
		}
	}
	if after.exists {
		states[path] = after
	} else {
		deleteStateTree(states, path, before.directory)
	}

	current, exists := pending[path]
	if !exists {
		if len(pending) >= o.config.MaxPending {
			dropped.Add(1)
			return
		}
		current = &pendingChange{path: path, before: before, first: item.at}
		pending[path] = current
	}
	current.op |= item.change.op
	current.after = after
	current.last = item.at
	current.count++
}

func observedPath(root, candidate string, matcher *project.IgnoreMatcher) (string, bool) {
	path, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	path = filepath.Clean(path)
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	if relative == "." || matcher.Ignored(relative, false) {
		return "", false
	}
	return path, true
}

func isIgnoreFile(path string) bool {
	base := filepath.Base(path)
	return base == ".gitignore" || base == ".aavignore"
}

func readyChanges(pending map[string]*pendingChange, now time.Time, debounce, renameWindow time.Duration, all bool, limit int) []*pendingChange {
	latestRenameChange := time.Time{}
	for _, item := range pending {
		if needsRenameWindow(item) && item.last.After(latestRenameChange) {
			latestRenameChange = item.last
		}
	}
	keys := make([]string, 0, len(pending))
	for path, item := range pending {
		age := now.Sub(item.last)
		required := debounce
		if needsRenameWindow(item) {
			age = now.Sub(latestRenameChange)
			required = max(required, renameWindow)
		}
		if all || age >= required {
			keys = append(keys, path)
		}
	}
	sort.Strings(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}
	result := make([]*pendingChange, 0, len(keys))
	for _, path := range keys {
		result = append(result, pending[path])
		delete(pending, path)
	}
	return result
}

func needsRenameWindow(item *pendingChange) bool {
	return item.op&(opRename|opRemove) != 0 || item.op&opCreate != 0 && !item.before.exists
}

func (o *Observer) eventsForBatch(ctx context.Context, observation wrapper.Observation, root string, batch []*pendingChange, sequence *uint64) []protocol.Event {
	if len(batch) == 0 {
		return nil
	}
	paths := make([]string, 0, len(batch))
	for _, item := range batch {
		paths = append(paths, item.path)
	}
	evidence := GitEvidence{}
	if o.config.Git != nil {
		evidence, _ = o.config.Git.Inspect(ctx, root, paths)
	}
	renamed, consumed := correlateRenames(root, batch, evidence)
	events := make([]protocol.Event, 0, len(batch))
	for _, pair := range renamed {
		*sequence++
		events = append(events, renameEvent(observation, *sequence, pair))
	}
	for _, item := range batch {
		if consumed[item.path] {
			continue
		}
		event, ok := changeEvent(observation, root, item, evidence)
		if !ok {
			continue
		}
		*sequence++
		event.EventID = fmt.Sprintf("%s:filesystem:%08d", observation.SessionID, *sequence)
		events = append(events, event)
	}
	return events
}

type renamePair struct {
	old *pendingChange
	new *pendingChange
}

func correlateRenames(root string, batch []*pendingChange, evidence GitEvidence) ([]renamePair, map[string]bool) {
	byRelative := make(map[string]*pendingChange, len(batch))
	creates := make([]*pendingChange, 0)
	olds := make([]*pendingChange, 0)
	for _, item := range batch {
		relative, _ := filepath.Rel(root, item.path)
		byRelative[filepath.ToSlash(relative)] = item
		if item.after.exists && item.op&opCreate != 0 {
			creates = append(creates, item)
		}
		if !item.after.exists && item.op&opRename != 0 {
			olds = append(olds, item)
		}
	}
	consumed := make(map[string]bool)
	pairs := make([]renamePair, 0)
	for oldRelative, newRelative := range evidence.Renames {
		old, oldOK := byRelative[oldRelative]
		newItem, newOK := byRelative[newRelative]
		if oldOK && newOK && !consumed[old.path] && !consumed[newItem.path] {
			pairs = append(pairs, renamePair{old: old, new: newItem})
			consumed[old.path], consumed[newItem.path] = true, true
		}
	}
	for _, old := range olds {
		if consumed[old.path] || old.before.directory {
			continue
		}
		var match *pendingChange
		for _, candidate := range creates {
			if consumed[candidate.path] || !sameFingerprint(old.before, candidate.after) {
				continue
			}
			if match != nil {
				match = nil
				break
			}
			match = candidate
		}
		if match != nil {
			pairs = append(pairs, renamePair{old: old, new: match})
			consumed[old.path], consumed[match.path] = true, true
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].old.path < pairs[j].old.path })
	return pairs, consumed
}

func sameFingerprint(left, right fileState) bool {
	return left.exists && right.exists &&
		left.directory == right.directory &&
		left.size == right.size &&
		left.modTime == right.modTime
}

func changeEvent(observation wrapper.Observation, root string, item *pendingChange, evidence GitEvidence) (protocol.Event, bool) {
	isDirectory := item.after.directory
	if !item.after.exists {
		isDirectory = item.before.directory
	}
	var eventType protocol.EventType
	var operationName string
	switch {
	case item.after.exists && !item.before.exists:
		if isDirectory {
			eventType = protocol.EventDirectoryCreated
		} else {
			eventType = protocol.EventFileCreated
		}
		operationName = "create"
	case !item.after.exists && item.before.exists:
		if isDirectory {
			eventType = protocol.EventDirectoryDeleted
		} else {
			eventType = protocol.EventFileDeleted
		}
		operationName = "delete"
	case item.after.exists && item.op&(opWrite|opCreate) != 0:
		if isDirectory {
			return protocol.Event{}, false
		}
		eventType = protocol.EventFileModified
		operationName = "modify"
	default:
		return protocol.Event{}, false
	}
	event := baseFilesystemEvent(observation, eventType, operationName, item.path, item.last, isDirectory)
	event.BytesBefore = bytePointer(item.before)
	event.BytesAfter = bytePointer(item.after)
	event.Metadata = map[string]interface{}{
		"coalesced_events": item.count,
		"evidence":         "filesystem_notification",
	}
	relative, _ := filepath.Rel(root, item.path)
	relative = filepath.ToSlash(relative)
	if status, ok := evidence.Status[relative]; ok {
		event.Metadata["git_status"] = string([]byte{status.Index, status.Worktree})
	}
	if delta, ok := evidence.Deltas[relative]; ok && !isDirectory {
		event.Metadata["work_source"] = "git_numstat"
		event.Metadata["work_confidence"] = string(protocol.ConfidenceCorrelated)
		if delta.Binary {
			value := true
			event.IsBinary = &value
		} else {
			added, deleted := delta.Added, delta.Deleted
			event.LinesAdded, event.LinesDeleted = &added, &deleted
		}
	}
	return event, true
}

func renameEvent(observation wrapper.Observation, sequence uint64, pair renamePair) protocol.Event {
	isDirectory := pair.old.before.directory
	eventType := protocol.EventFileRenamed
	if isDirectory {
		eventType = protocol.EventDirectoryRenamed
	} else if filepath.Dir(pair.old.path) != filepath.Dir(pair.new.path) {
		eventType = protocol.EventFileMoved
	}
	event := baseFilesystemEvent(observation, eventType, "rename", pair.new.path, pair.new.last, isDirectory)
	event.EventID = fmt.Sprintf("%s:filesystem:%08d", observation.SessionID, sequence)
	event.PreviousPath = pair.old.path
	event.BytesBefore = bytePointer(pair.old.before)
	event.BytesAfter = bytePointer(pair.new.after)
	event.SourceConfidence = protocol.ConfidenceCorrelated
	event.Metadata = map[string]interface{}{
		"evidence": "unique_filesystem_or_git_correlation",
	}
	return event
}

func baseFilesystemEvent(observation wrapper.Observation, eventType protocol.EventType, operationName, path string, at time.Time, directory bool) protocol.Event {
	isDirectory := directory
	return protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		SessionID:        observation.SessionID,
		AgentType:        observation.AgentType,
		AdapterID:        wrapper.AdapterID,
		AdapterVersion:   wrapper.AdapterVersion,
		SourceType:       protocol.SourceFilesystem,
		SourceConfidence: protocol.ConfidenceObserved,
		EventType:        eventType,
		Operation:        operationName,
		Timestamp:        at.UTC(),
		ProcessID:        observation.ProcessID,
		ProjectRoot:      observation.ProjectRoot,
		Path:             path,
		IsDirectory:      &isDirectory,
	}
}

func droppedEvent(observation wrapper.Observation, sequence uint64, at time.Time, count uint64) protocol.Event {
	event := baseFilesystemEvent(observation, protocol.EventDroppedOrCoalesced, "drop", "", at, false)
	event.EventID = fmt.Sprintf("%s:filesystem:%08d", observation.SessionID, sequence)
	event.SourceConfidence = protocol.ConfidenceInferred
	event.IsDirectory = nil
	event.Metadata = map[string]interface{}{
		"dropped_events": count,
		"reason":         "filesystem_queue_or_pending_capacity",
	}
	return event
}

func statState(path string) fileState {
	info, err := os.Lstat(path)
	if err != nil {
		return fileState{}
	}
	return stateFromInfo(info)
}

func stateFromInfo(info os.FileInfo) fileState {
	return fileState{
		exists:    true,
		directory: info.IsDir(),
		size:      info.Size(),
		modTime:   info.ModTime().UnixNano(),
	}
}

func bytePointer(state fileState) *int64 {
	if !state.exists || state.directory {
		return nil
	}
	value := state.size
	return &value
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func deleteStateTree(states map[string]fileState, path string, directory bool) {
	delete(states, path)
	if !directory {
		return
	}
	prefix := path + string(filepath.Separator)
	for candidate := range states {
		if strings.HasPrefix(candidate, prefix) {
			delete(states, candidate)
		}
	}
}

var _ wrapper.FallbackObserver = (*Observer)(nil)
