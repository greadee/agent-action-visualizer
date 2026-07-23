package main

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/greadee/agent-action-visualizer/internal/activity"
	"github.com/greadee/agent-action-visualizer/internal/graph"
	"github.com/greadee/agent-action-visualizer/internal/ingest"
	"github.com/greadee/agent-action-visualizer/internal/project"
	"github.com/greadee/agent-action-visualizer/internal/session"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	mu       sync.Mutex
	root     string
	identity *graph.IdentityRegistry
	snapshot graph.GraphSnapshot
	focus    *session.Engine
}

func NewApp() *App {
	return &App{focus: session.NewEngine(2 * time.Minute)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Health reports the embedded collector shell state without network access.
func (a *App) Health() map[string]string {
	return map[string]string{
		"service": "agent-action-visualizer",
		"status":  "ready",
	}
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

type trailAccessDTO struct {
	Sequence   int                 `json:"sequence"`
	NodeID     string              `json:"node_id,omitempty"`
	Path       string              `json:"path"`
	StartedAt  time.Time           `json:"started_at"`
	EndedAt    *time.Time          `json:"ended_at,omitempty"`
	DurationMS int64               `json:"duration_ms"`
	Operations []string            `json:"operations"`
	Source     protocol.SourceType `json:"source"`
	Confidence protocol.Confidence `json:"confidence"`
	AgentID    string              `json:"agent_id,omitempty"`
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
	nodes := a.identity.Assign(enriched)
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
	result := a.liveFocusDTO(state)
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "aav:focus", result)
	}
	return result, nil
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
	return ""
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
			AgentID: interval.AgentID,
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
	next := graph.Layout(a.identity.Assign(enriched), a.snapshot.Revision+1)
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
