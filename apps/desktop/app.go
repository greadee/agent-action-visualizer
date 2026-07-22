package main

import (
	"context"
	"errors"
	"runtime"
	"sync"

	"github.com/greadee/agent-action-visualizer/internal/activity"
	"github.com/greadee/agent-action-visualizer/internal/graph"
	"github.com/greadee/agent-action-visualizer/internal/project"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	mu       sync.Mutex
	root     string
	identity *graph.IdentityRegistry
	snapshot graph.GraphSnapshot
}

func NewApp() *App {
	return &App{}
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
