package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/greadee/agent-action-visualizer/internal/activity"
	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	"github.com/greadee/agent-action-visualizer/internal/graph"
	"github.com/greadee/agent-action-visualizer/internal/ingest"
	localipc "github.com/greadee/agent-action-visualizer/internal/ipc"
	"github.com/greadee/agent-action-visualizer/internal/project"
	"github.com/greadee/agent-action-visualizer/internal/replay"
	"github.com/greadee/agent-action-visualizer/internal/session"
	"github.com/greadee/agent-action-visualizer/internal/store"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx             context.Context
	mu              sync.Mutex
	root            string
	identity        *graph.IdentityRegistry
	snapshot        graph.GraphSnapshot
	focus           *session.Engine
	diffs           *workdiff.Pipeline
	diffDone        chan struct{}
	events          *ingest.Collector
	dedupe          *ingest.Deduper
	ipc             *localipc.Server
	ipcEndpoint     string
	collectorStatus string
	journal         *store.Store
	journalQueue    chan journalRecord
	journalDone     chan struct{}
	journalStatus   string
}

type journalRecord struct {
	event     protocol.Event
	state     session.State
	projectID string
}

func NewApp() *App {
	app := &App{
		focus:    session.NewEngine(2 * time.Minute),
		diffs:    workdiff.NewPipeline(64, 16, workdiff.NewCommandGitRunner(2*time.Second)),
		diffDone: make(chan struct{}),
		dedupe:   ingest.NewDeduper(10 * time.Minute),
	}
	app.events = ingest.NewCollector(256, func(_ context.Context, event protocol.Event) {
		if app.dedupe.Duplicate(event, time.Now()) {
			return
		}
		_, _ = app.PublishActivityEvent(event)
	})
	app.events.Start(context.Background())
	go app.consumeDiffResults()
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startJournal()
	a.startCollector()
}

func (a *App) shutdown(context.Context) {
	if a.ipc != nil {
		a.ipc.Close()
	}
	a.events.Stop()
	a.diffs.Close()
	<-a.diffDone
	a.stopJournal()
}

// Health reports the embedded collector shell state without network access.
func (a *App) Health() map[string]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]string{
		"service":     "agent-action-visualizer",
		"status":      "ready",
		"collector":   a.collectorStatus,
		"persistence": a.journalStatus,
	}
}

func (a *App) startJournal() {
	directory, err := os.UserConfigDir()
	if err != nil {
		a.journalStatus = "unavailable"
		return
	}
	if err := os.MkdirAll(filepath.Join(directory, "agent-action-visualizer"), 0o700); err != nil {
		a.journalStatus = "unavailable"
		return
	}
	a.startJournalAt(filepath.Join(directory, "agent-action-visualizer", "sessions.db"))
}

func (a *App) startJournalAt(path string) {
	journal, err := store.Open(path)
	if err != nil {
		a.journalStatus = "unavailable"
		return
	}
	a.mu.Lock()
	if a.journal != nil {
		a.mu.Unlock()
		_ = journal.Close()
		return
	}
	a.journal, a.journalQueue, a.journalDone = journal, make(chan journalRecord, 256), make(chan struct{})
	a.journalStatus = "ready"
	queue, done := a.journalQueue, a.journalDone
	a.mu.Unlock()
	go func() {
		defer close(done)
		for record := range queue {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			if record.projectID != "" {
				_ = journal.EnsureProject(ctx, record.projectID, record.projectID)
			}
			_ = journal.SaveEvent(ctx, record.event)
			_ = journal.SaveSession(ctx, record.state, record.projectID)
			cancel()
		}
	}()
}

func (a *App) stopJournal() {
	a.mu.Lock()
	journal, queue, done := a.journal, a.journalQueue, a.journalDone
	a.journal, a.journalQueue, a.journalDone = nil, nil, nil
	if journal != nil {
		a.journalStatus = "stopped"
	}
	a.mu.Unlock()
	if queue != nil {
		close(queue)
		<-done
	}
	if journal != nil {
		_ = journal.Close()
	}
}

// queueJournal never blocks lifecycle observation; a saturated or unavailable
// local journal only loses replay history, not agent behavior.
func (a *App) queueJournal(event protocol.Event, state session.State) {
	if a.journalQueue == nil {
		return
	}
	select {
	case a.journalQueue <- journalRecord{event: event, state: state, projectID: a.root}:
	default:
		a.journalStatus = "degraded"
	}
}

func (a *App) startCollector() {
	a.mu.Lock()
	defer a.mu.Unlock()
	endpoint := a.ipcEndpoint
	if endpoint == "" {
		endpoint = localipc.DefaultEndpoint()
	}
	server := localipc.NewServer(endpoint, a.events.Submit)
	if err := server.Start(); err != nil {
		a.collectorStatus = "unavailable"
		return
	}
	a.ipc = server
	a.collectorStatus = "ready"
}

type graphNodeDTO struct {
	ID       string               `json:"id"`
	Path     string               `json:"path"`
	Kind     project.NodeKind     `json:"kind"`
	Position [3]float64           `json:"position"`
	Activity project.NodeActivity `json:"activity"`
}
type graphEdgeDTO struct {
	Source string `json:"source"`
	Target string `json:"target"`
}
type graphSnapshotDTO struct {
	Revision int64          `json:"revision"`
	Nodes    []graphNodeDTO `json:"nodes"`
	Edges    []graphEdgeDTO `json:"edges"`
}
type graphPatchDTO struct {
	Revision int64          `json:"revision"`
	Added    []graphNodeDTO `json:"added"`
	Updated  []graphNodeDTO `json:"updated"`
	Removed  []string       `json:"removed"`
	Edges    []graphEdgeDTO `json:"edges"`
}

type liveFocusDTO struct {
	SessionID        string              `json:"session_id"`
	ActiveNodeID     string              `json:"active_node_id,omitempty"`
	ActivePath       string              `json:"active_path,omitempty"`
	PreviousNodeID   string              `json:"previous_node_id,omitempty"`
	PreviousPath     string              `json:"previous_path,omitempty"`
	SecondaryNodeIDs []string            `json:"secondary_node_ids,omitempty"`
	SecondaryPaths   []string            `json:"secondary_paths,omitempty"`
	Operation        string              `json:"operation,omitempty"`
	Source           protocol.SourceType `json:"source,omitempty"`
	Confidence       protocol.Confidence `json:"confidence,omitempty"`
	Timestamp        time.Time           `json:"timestamp,omitempty"`
	Trail            []trailAccessDTO    `json:"trail"`
}

type sessionSummaryDTO struct {
	ID        string     `json:"id"`
	StartedAt time.Time  `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	Status    string     `json:"status"`
}

type replayTimelineDTO struct {
	Index     int                `json:"index"`
	Timestamp time.Time          `json:"timestamp"`
	EventType protocol.EventType `json:"event_type"`
	Path      string             `json:"path,omitempty"`
	IsAccess  bool               `json:"is_access"`
}

type replaySessionDTO struct {
	Session    sessionSummaryDTO   `json:"session"`
	Cursor     int                 `json:"cursor"`
	EventCount int                 `json:"event_count"`
	CursorAt   time.Time           `json:"cursor_at,omitempty"`
	Focus      liveFocusDTO        `json:"focus"`
	Timeline   []replayTimelineDTO `json:"timeline"`
}

type trailAccessDTO struct {
	Sequence       int                 `json:"sequence"`
	NodeID         string              `json:"node_id,omitempty"`
	Path           string              `json:"path"`
	StartedAt      time.Time           `json:"started_at"`
	EndedAt        *time.Time          `json:"ended_at,omitempty"`
	DurationMS     int64               `json:"duration_ms"`
	Operations     []string            `json:"operations"`
	Source         protocol.SourceType `json:"source"`
	Confidence     protocol.Confidence `json:"confidence"`
	AgentID        string              `json:"agent_id,omitempty"`
	LinesAdded     *int64              `json:"lines_added,omitempty"`
	LinesDeleted   *int64              `json:"lines_deleted,omitempty"`
	WorkStatus     workdiff.Status     `json:"work_status,omitempty"`
	WorkSource     workdiff.Source     `json:"work_source,omitempty"`
	WorkConfidence protocol.Confidence `json:"work_confidence,omitempty"`
}

// LoadProject scans metadata only, initializes stable identity, and publishes a full graph snapshot.
func (a *App) LoadProject(root string) (graphSnapshotDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	scanned, err := project.NewScanner().Scan(context.Background(), root)
	if err != nil {
		return graphSnapshotDTO{}, err
	}
	a.root = scanned.Root
	a.identity = graph.NewIdentityRegistry(scanned.Root, runtime.GOOS == "windows")
	a.focus = session.NewEngine(2 * time.Minute)
	enriched, err := activity.Enrich(context.Background(), scanned.Root, scanned.Nodes)
	if err != nil {
		return graphSnapshotDTO{}, err
	}
	nodes := a.withTombstones(a.identity.Assign(enriched))
	a.snapshot = graph.Layout(nodes, 1)
	result := snapshotDTO(a.snapshot)
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "aav:graph:snapshot", result)
	}
	return result, nil
}

// PublishActivityEvent accepts a normalized protocol event, updates session
// focus, and publishes a compact state for camera and graph consumers.
func (a *App) PublishActivityEvent(event protocol.Event) (liveFocusDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.root == "" || a.identity == nil {
		return liveFocusDTO{}, errors.New("no project is loaded")
	}
	normalized, err := ingest.Normalize(event, a.root)
	if err != nil {
		return liveFocusDTO{}, err
	}
	a.applyIdentityLifecycle(normalized)
	if normalized.Path != "" && normalized.NodeID == "" {
		normalized.NodeID = a.nodeIDForPath(normalized.Path)
	}
	if normalized.Metadata != nil {
		paths, err := a.normalizeSecondaryPaths(normalized.Metadata["secondary_paths"])
		if err != nil {
			return liveFocusDTO{}, err
		}
		normalized.Metadata["secondary_paths"] = paths
	}
	state, err := a.focus.Apply(normalized)
	if err != nil {
		return liveFocusDTO{}, err
	}
	state = a.submitDiff(normalized, state)
	a.queueJournal(normalized, state)
	result := a.liveFocusDTO(state)
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "aav:focus", result)
	}
	return result, nil
}

func (a *App) submitDiff(event protocol.Event, state session.State) session.State {
	if !workEvent(event.EventType) || event.Path == "" || len(state.Intervals) == 0 {
		return state
	}
	interval := state.Intervals[len(state.Intervals)-1]
	if interval.Path != event.Path {
		return state
	}
	isBinary := event.IsBinary != nil && *event.IsBinary
	outcome := a.diffs.Offer(workdiff.Request{
		Key: event.EventID, SessionID: event.SessionID, Sequence: interval.Sequence,
		ProjectRoot: a.root, Path: event.Path, StructuredAdded: event.LinesAdded,
		StructuredDeleted: event.LinesDeleted, IsBinary: isBinary,
		Confidence: event.SourceConfidence,
	})
	if outcome == workdiff.SubmitDuplicate {
		return state
	}
	if outcome == workdiff.SubmitAccepted {
		pending, err := a.focus.SetWorkResult(
			event.SessionID, interval.Sequence, event.LinesAdded, event.LinesDeleted,
			workdiff.StatusPending, workdiff.SourceUnknown, protocol.ConfidenceInferred,
		)
		if err == nil {
			return pending
		}
		return state
	}
	fallback, err := a.focus.SetWorkResult(
		event.SessionID, interval.Sequence, nil, nil, workdiff.StatusUnknown,
		workdiff.SourceUnknown, protocol.ConfidenceInferred,
	)
	if err == nil {
		return fallback
	}
	return state
}

func workEvent(eventType protocol.EventType) bool {
	switch eventType {
	case protocol.EventFileCreated, protocol.EventFileModified,
		protocol.EventFilePatched, protocol.EventFileDeleted:
		return true
	default:
		return false
	}
}

func (a *App) consumeDiffResults() {
	defer close(a.diffDone)
	for result := range a.diffs.Results() {
		a.mu.Lock()
		state, err := a.focus.SetWorkResult(
			result.SessionID, result.Sequence, result.LinesAdded, result.LinesDeleted,
			result.Status, result.Source, result.Confidence,
		)
		if err == nil && a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "aav:focus", a.liveFocusDTO(state))
		}
		if err == nil {
			event := protocol.Event{SchemaVersion: protocol.SchemaVersion, EventID: "diff:" + result.Key, SessionID: result.SessionID, SourceType: protocol.SourceGit, SourceConfidence: result.Confidence, EventType: protocol.EventDiffCalculated, Timestamp: time.Now().UTC(), Status: string(result.Status), Operation: string(result.Source), LinesAdded: result.LinesAdded, LinesDeleted: result.LinesDeleted, Metadata: map[string]interface{}{"access_sequence": result.Sequence}}
			a.queueJournal(event, state)
		}
		a.mu.Unlock()
	}
}

// ListPersistedSessions exposes current-project session metadata only.
func (a *App) ListPersistedSessions() ([]sessionSummaryDTO, error) {
	a.mu.Lock()
	journal, root := a.journal, a.root
	a.mu.Unlock()
	if journal == nil || root == "" {
		return []sessionSummaryDTO{}, nil
	}
	items, err := journal.Sessions(context.Background(), root)
	if err != nil {
		return nil, err
	}
	result := make([]sessionSummaryDTO, 0, len(items))
	for _, item := range items {
		result = append(result, sessionSummaryDTO{ID: item.ID, StartedAt: item.StartedAt, StoppedAt: item.StoppedAt, Status: item.Status})
	}
	return result, nil
}

// ReplayPersistedSession rebuilds the selected session at a stable event cursor.
func (a *App) ReplayPersistedSession(sessionID string, cursor int) (replaySessionDTO, error) {
	a.mu.Lock()
	journal, root := a.journal, a.root
	a.mu.Unlock()
	if journal == nil || root == "" {
		return replaySessionDTO{}, errors.New("persisted session history is unavailable")
	}
	items, err := journal.Sessions(context.Background(), root)
	if err != nil {
		return replaySessionDTO{}, err
	}
	var summary sessionSummaryDTO
	for _, item := range items {
		if item.ID == sessionID {
			summary = sessionSummaryDTO{ID: item.ID, StartedAt: item.StartedAt, StoppedAt: item.StoppedAt, Status: item.Status}
			break
		}
	}
	if summary.ID == "" {
		return replaySessionDTO{}, errors.New("session is not available for the loaded project")
	}
	events, err := journal.Events(context.Background(), sessionID)
	if err != nil {
		return replaySessionDTO{}, err
	}
	snapshot, err := replay.Reconstruct(events, cursor, 2*time.Minute)
	if err != nil {
		return replaySessionDTO{}, err
	}
	a.mu.Lock()
	focus := a.liveFocusDTO(snapshot.State)
	a.mu.Unlock()
	timeline := make([]replayTimelineDTO, 0, len(events))
	for index, event := range events {
		timeline = append(timeline, replayTimelineDTO{Index: index, Timestamp: event.Timestamp, EventType: event.EventType, Path: event.Path, IsAccess: replayAccessEvent(event.EventType)})
	}
	return replaySessionDTO{Session: summary, Cursor: snapshot.Cursor, EventCount: snapshot.EventCount, CursorAt: snapshot.CursorAt, Focus: focus, Timeline: timeline}, nil
}

func replayAccessEvent(eventType protocol.EventType) bool {
	switch eventType {
	case protocol.EventFileFocused, protocol.EventFileRead, protocol.EventFileCreated, protocol.EventFileModified, protocol.EventFilePatched, protocol.EventFileRenamed, protocol.EventFileMoved, protocol.EventFileDeleted:
		return true
	default:
		return false
	}
}

func (a *App) normalizeSecondaryPaths(value interface{}) ([]string, error) {
	var values []string
	switch raw := value.(type) {
	case nil:
		return nil, nil
	case []string:
		values = raw
	case []interface{}:
		for _, item := range raw {
			path, ok := item.(string)
			if !ok {
				return nil, errors.New("secondary_paths must contain only strings")
			}
			values = append(values, path)
		}
	default:
		return nil, errors.New("secondary_paths must be an array")
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		path, err := ingest.NormalizeProjectPath(a.root, value)
		if err != nil {
			return nil, errors.New("secondary path escapes selected project")
		}
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result, nil
}

func (a *App) nodeIDForPath(path string) string {
	for _, node := range a.snapshot.Nodes {
		if node.Path == path {
			return node.ID
		}
	}
	if a.identity != nil {
		if node, ok := a.identity.Lookup(path); ok {
			return node.ID
		}
	}
	return ""
}

func (a *App) applyIdentityLifecycle(event protocol.Event) {
	if a.identity == nil {
		return
	}
	switch event.EventType {
	case protocol.EventFileRenamed, protocol.EventFileMoved:
		if event.PreviousPath != "" && event.Path != "" {
			_, _ = a.identity.Rename(event.PreviousPath, event.Path)
		}
	case protocol.EventFileDeleted:
		if event.Path != "" {
			_, _ = a.identity.Delete(event.Path)
		}
	}
}

func (a *App) withTombstones(nodes []project.Node) []project.Node {
	if a.identity == nil {
		return nodes
	}
	seen := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		seen[node.Path] = true
	}
	for _, node := range a.identity.Tombstones() {
		if !seen[node.Path] {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (a *App) liveFocusDTO(state session.State) liveFocusDTO {
	secondaryIDs := make([]string, 0, len(state.SecondaryPaths))
	for _, path := range state.SecondaryPaths {
		if id := a.nodeIDForPath(path); id != "" && id != state.ActiveNodeID {
			secondaryIDs = append(secondaryIDs, id)
		}
	}
	return liveFocusDTO{
		SessionID: state.SessionID, ActiveNodeID: state.ActiveNodeID,
		ActivePath: state.ActivePath, PreviousNodeID: state.PreviousNodeID,
		PreviousPath: state.PreviousPath, SecondaryNodeIDs: secondaryIDs,
		SecondaryPaths: state.SecondaryPaths, Operation: state.ActiveOperation,
		Source: state.ActiveSource, Confidence: state.ActiveConfidence,
		Timestamp: state.ActiveTimestamp, Trail: a.trailDTOs(state.Intervals),
	}
}

func (a *App) trailDTOs(intervals []session.AccessInterval) []trailAccessDTO {
	result := make([]trailAccessDTO, 0, len(intervals))
	for _, interval := range intervals {
		nodeID := interval.NodeID
		if nodeID == "" {
			nodeID = a.nodeIDForPath(interval.Path)
		}
		result = append(result, trailAccessDTO{
			Sequence: interval.Sequence, NodeID: nodeID, Path: interval.Path,
			StartedAt: interval.StartedAt, EndedAt: interval.EndedAt,
			DurationMS: interval.Duration.Milliseconds(),
			Operations: append([]string(nil), interval.Operations...),
			Source:     interval.Source, Confidence: interval.Confidence,
			AgentID: interval.AgentID, LinesAdded: interval.LinesAdded,
			LinesDeleted: interval.LinesDeleted, WorkStatus: interval.WorkStatus,
			WorkSource: interval.WorkSource, WorkConfidence: interval.WorkConfidence,
		})
	}
	return result
}

// RefreshProject emits only graph changes after the initial snapshot.
func (a *App) RefreshProject() (graphPatchDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.root == "" || a.identity == nil {
		return graphPatchDTO{}, errors.New("no project is loaded")
	}
	scanned, err := project.NewScanner().Scan(context.Background(), a.root)
	if err != nil {
		return graphPatchDTO{}, err
	}
	enriched, err := activity.Enrich(context.Background(), scanned.Root, scanned.Nodes)
	if err != nil {
		return graphPatchDTO{}, err
	}
	next := graph.Layout(a.withTombstones(a.identity.Assign(enriched)), a.snapshot.Revision+1)
	change := graph.Diff(a.snapshot, next)
	a.snapshot = next
	result := patchDTO(change)
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "aav:graph:patch", result)
	}
	return result, nil
}

func snapshotDTO(value graph.GraphSnapshot) graphSnapshotDTO {
	return graphSnapshotDTO{Revision: value.Revision, Nodes: nodeDTOs(value.Nodes), Edges: edgeDTOs(value.Edges)}
}
func patchDTO(value graph.Patch) graphPatchDTO {
	return graphPatchDTO{Revision: value.Revision, Added: nodeDTOs(value.Added), Updated: nodeDTOs(value.Updated), Removed: value.Removed, Edges: edgeDTOs(value.Edges)}
}
func nodeDTOs(nodes []graph.PositionedNode) []graphNodeDTO {
	result := make([]graphNodeDTO, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, graphNodeDTO{ID: node.ID, Path: node.Path, Kind: node.Kind, Position: [3]float64{node.Position.X, node.Position.Y, node.Position.Z}, Activity: node.Activity})
	}
	return result
}
func edgeDTOs(edges []graph.Edge) []graphEdgeDTO {
	result := make([]graphEdgeDTO, 0, len(edges))
	for _, edge := range edges {
		result = append(result, graphEdgeDTO{Source: edge.Source, Target: edge.Target})
	}
	return result
}
