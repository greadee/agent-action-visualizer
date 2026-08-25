// Package diff calculates line deltas outside the event-submission path.
package diff

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type Status string

const (
	StatusKnown               Status = "known"
	StatusEmpty               Status = "empty"
	StatusUnknown             Status = "unknown"
	StatusBinary              Status = "binary"
	StatusUnsupportedEncoding Status = "unsupported_encoding"
	StatusPending             Status = "pending"
)

type Source string

const (
	SourceStructuredPatch    Source = "structured_patch"
	SourceCorrelatedSnapshot Source = "correlated_snapshot"
	SourceGitNumstat         Source = "git_numstat"
	SourceUnknown            Source = "unknown"
)

type Request struct {
	Key                      string
	SessionID                string
	Sequence                 int
	ProjectRoot              string
	Path                     string
	StructuredAdded          *int64
	StructuredDeleted        *int64
	StructuredPatch          *string
	Before                   []byte
	After                    []byte
	AllowSnapshotCorrelation bool
	IsBinary                 bool
	Confidence               protocol.Confidence
}

type Result struct {
	Key          string
	SessionID    string
	Sequence     int
	Path         string
	LinesAdded   *int64
	LinesDeleted *int64
	Status       Status
	Source       Source
	Confidence   protocol.Confidence
}

type GitDelta struct {
	Added   int64
	Deleted int64
	Binary  bool
}

type GitRunner interface {
	Numstat(context.Context, string, []string) (map[string]GitDelta, error)
}

type Stats struct {
	Accepted   uint64
	Dropped    uint64
	Duplicates uint64
	Completed  uint64
}

type SubmitOutcome string

const (
	SubmitAccepted  SubmitOutcome = "accepted"
	SubmitDuplicate SubmitOutcome = "duplicate"
	SubmitDropped   SubmitOutcome = "dropped"
)

type Pipeline struct {
	ctx       context.Context
	cancel    context.CancelFunc
	queue     chan Request
	results   chan Result
	done      chan struct{}
	runner    GitRunner
	batchSize int

	mu        sync.Mutex
	pending   map[string]struct{}
	recent    map[string]struct{}
	order     []string
	maxRecent int
	stats     Stats
	once      sync.Once
}

func NewPipeline(capacity, batchSize int, runner GitRunner) *Pipeline {
	if capacity < 1 {
		capacity = 1
	}
	if batchSize < 1 {
		batchSize = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pipeline{
		ctx: ctx, cancel: cancel, queue: make(chan Request, capacity),
		results: make(chan Result, capacity*2), done: make(chan struct{}),
		runner: runner, batchSize: batchSize, pending: make(map[string]struct{}),
		recent: make(map[string]struct{}), maxRecent: max(64, capacity*4),
	}
	go p.run()
	return p
}

func (p *Pipeline) Results() <-chan Result { return p.results }

func (p *Pipeline) Submit(request Request) bool {
	return p.Offer(request) == SubmitAccepted
}

func (p *Pipeline) Offer(request Request) SubmitOutcome {
	if request.Key == "" {
		return SubmitDropped
	}
	request.Before = append([]byte(nil), request.Before...)
	request.After = append([]byte(nil), request.After...)
	p.mu.Lock()
	_, pending := p.pending[request.Key]
	_, recent := p.recent[request.Key]
	if pending || recent {
		p.stats.Duplicates++
		p.mu.Unlock()
		return SubmitDuplicate
	}
	p.pending[request.Key] = struct{}{}
	p.mu.Unlock()

	select {
	case <-p.ctx.Done():
		p.mu.Lock()
		delete(p.pending, request.Key)
		p.stats.Dropped++
		p.mu.Unlock()
		return SubmitDropped
	case p.queue <- request:
		p.mu.Lock()
		p.stats.Accepted++
		p.mu.Unlock()
		return SubmitAccepted
	default:
		p.mu.Lock()
		delete(p.pending, request.Key)
		p.stats.Dropped++
		p.mu.Unlock()
		return SubmitDropped
	}
}

func (p *Pipeline) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stats
}

func (p *Pipeline) Close() {
	p.once.Do(func() {
		p.cancel()
		<-p.done
	})
}

func (p *Pipeline) run() {
	defer close(p.done)
	defer close(p.results)
	for {
		select {
		case <-p.ctx.Done():
			return
		case request := <-p.queue:
			batch := []Request{request}
			timer := time.NewTimer(2 * time.Millisecond)
		collect:
			for len(batch) < p.batchSize {
				select {
				case request = <-p.queue:
					batch = append(batch, request)
				case <-timer.C:
					break collect
				case <-p.ctx.Done():
					timer.Stop()
					return
				}
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			p.process(batch)
		}
	}
}

func (p *Pipeline) process(batch []Request) {
	gitGroups := make(map[string][]Request)
	for _, request := range batch {
		if result, resolved := resolveDirect(request); resolved {
			p.publish(result)
			continue
		}
		if request.ProjectRoot == "" || request.Path == "" || p.runner == nil {
			p.publish(unknownResult(request))
			continue
		}
		gitGroups[request.ProjectRoot] = append(gitGroups[request.ProjectRoot], request)
	}
	for root, requests := range gitGroups {
		paths := make([]string, 0, len(requests))
		seen := make(map[string]bool, len(requests))
		for _, request := range requests {
			if !seen[request.Path] {
				seen[request.Path] = true
				paths = append(paths, request.Path)
			}
		}
		values, err := p.runner.Numstat(p.ctx, root, paths)
		for _, request := range requests {
			if err != nil {
				p.publish(unknownResult(request))
				continue
			}
			value, ok := values[request.Path]
			if !ok {
				p.publish(unknownResult(request))
				continue
			}
			if value.Binary {
				p.publish(newResult(request, nil, nil, StatusBinary, SourceGitNumstat, protocol.ConfidenceCorrelated))
				continue
			}
			added, deleted := value.Added, value.Deleted
			status := StatusKnown
			if added == 0 && deleted == 0 {
				status = StatusEmpty
			}
			p.publish(newResult(request, &added, &deleted, status, SourceGitNumstat, protocol.ConfidenceCorrelated))
		}
	}
}

func (p *Pipeline) publish(result Result) {
	select {
	case p.results <- result:
		p.mu.Lock()
		delete(p.pending, result.Key)
		p.rememberLocked(result.Key)
		p.stats.Completed++
		p.mu.Unlock()
	case <-p.ctx.Done():
	}
}

func (p *Pipeline) rememberLocked(key string) {
	if _, exists := p.recent[key]; exists {
		return
	}
	p.recent[key] = struct{}{}
	p.order = append(p.order, key)
	if len(p.order) <= p.maxRecent {
		return
	}
	oldest := p.order[0]
	p.order = p.order[1:]
	delete(p.recent, oldest)
}

func resolveDirect(request Request) (Result, bool) {
	if request.IsBinary {
		return newResult(request, nil, nil, StatusBinary, SourceStructuredPatch, request.Confidence), true
	}
	if request.StructuredAdded != nil || request.StructuredDeleted != nil {
		added, deleted := int64(0), int64(0)
		if request.StructuredAdded != nil {
			added = max(0, *request.StructuredAdded)
		}
		if request.StructuredDeleted != nil {
			deleted = max(0, *request.StructuredDeleted)
		}
		status := StatusKnown
		if added == 0 && deleted == 0 {
			status = StatusEmpty
		}
		return newResult(request, &added, &deleted, status, SourceStructuredPatch, request.Confidence), true
	}
	if request.StructuredPatch != nil {
		added, deleted, status := countPatch([]byte(*request.StructuredPatch))
		if status == StatusKnown || status == StatusEmpty {
			return newResult(request, &added, &deleted, status, SourceStructuredPatch, request.Confidence), true
		}
		return newResult(request, nil, nil, status, SourceStructuredPatch, request.Confidence), true
	}
	if request.AllowSnapshotCorrelation && (request.Before != nil || request.After != nil) {
		added, deleted, status := countSnapshots(request.Before, request.After)
		if status == StatusKnown || status == StatusEmpty {
			return newResult(request, &added, &deleted, status, SourceCorrelatedSnapshot, protocol.ConfidenceCorrelated), true
		}
		if status == StatusUnknown {
			return Result{}, false
		}
		return newResult(request, nil, nil, status, SourceCorrelatedSnapshot, protocol.ConfidenceCorrelated), true
	}
	return Result{}, false
}

func countPatch(patch []byte) (int64, int64, Status) {
	if bytes.IndexByte(patch, 0) >= 0 || bytes.Contains(patch, []byte("Binary files ")) {
		return 0, 0, StatusBinary
	}
	if !utf8.Valid(patch) {
		return 0, 0, StatusUnsupportedEncoding
	}
	var added, deleted int64
	inHunk := false
	for _, line := range strings.Split(strings.ReplaceAll(string(patch), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			continue
		}
		if !inHunk && (strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---")) {
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deleted++
		}
	}
	if added == 0 && deleted == 0 {
		return 0, 0, StatusEmpty
	}
	return added, deleted, StatusKnown
}

const maxSnapshotCells = 1_000_000

func countSnapshots(before, after []byte) (int64, int64, Status) {
	if bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 {
		return 0, 0, StatusBinary
	}
	if !utf8.Valid(before) || !utf8.Valid(after) {
		return 0, 0, StatusUnsupportedEncoding
	}
	left, right := splitLines(string(before)), splitLines(string(after))
	if len(left)*len(right) > maxSnapshotCells {
		return 0, 0, StatusUnknown
	}
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for _, leftLine := range left {
		for index, rightLine := range right {
			if leftLine == rightLine {
				current[index+1] = previous[index] + 1
			} else {
				current[index+1] = max(current[index], previous[index+1])
			}
		}
		previous, current = current, previous
		clear(current)
	}
	common := previous[len(right)]
	added, deleted := int64(len(right)-common), int64(len(left)-common)
	if added == 0 && deleted == 0 {
		return 0, 0, StatusEmpty
	}
	return added, deleted, StatusKnown
}

func splitLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func unknownResult(request Request) Result {
	return newResult(request, nil, nil, StatusUnknown, SourceUnknown, protocol.ConfidenceInferred)
}

func newResult(request Request, added, deleted *int64, status Status, source Source, confidence protocol.Confidence) Result {
	if confidence == "" {
		confidence = protocol.ConfidenceInferred
	}
	return Result{
		Key: request.Key, SessionID: request.SessionID, Sequence: request.Sequence,
		Path: request.Path, LinesAdded: added, LinesDeleted: deleted,
		Status: status, Source: source, Confidence: confidence,
	}
}
