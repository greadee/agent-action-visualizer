# Git involvement and activity geometry

## Visual contract

The project root stays at the origin. Every other project node is normalized onto a shell with radius `7.0`; directory depth no longer changes distance from the center.

Nodes are grouped by their most recent Git commit. Commit groups are ordered newest-first and their centers are distributed across the sphere using a deterministic golden-angle sequence. Nodes within a group receive a small deterministic tangent offset, so files touched by the same latest commit remain visually adjacent without overlapping at one coordinate.

The layout is deterministic for the same history. A new latest commit intentionally recomputes group centers so the visual grouping continues to describe current involvement rather than preserving stale coordinates.

File activity starts at the file node and extends radially outward. The endpoint is the edit point. Directories and the root do not receive edit-point extrusions.

For the selected Time or Work value `v`, relative to the largest visible value `max`, extrusion length is:

```text
length = 0.45 + 3.55 * log(1 + v) / log(1 + max)
```

Zero or missing agent-reported values produce no extrusion. This avoids presenting inferred Git data as elapsed time or work.

## Data authority

Git history provides:

- latest commit and timestamp;
- latest-event summary;
- commit-touch access count;
- latest-involvement cluster;
- `git commit` as an observed tool plus explicit `[tool:name]` commit annotations.

Import is bounded to the 5,000 most recent commits. Inspector access counts therefore describe commit touches inside that explicit history window. Agent report paths must be project-relative and all numeric work values must be non-negative.

Agents provide `.aav/activity-v1.json` with total duration, lines added/deleted, recent tools, and session identifiers. A report has this shape:

```json
{
  "schema_version": 1,
  "files": {
    "src/ui/Graph.tsx": {
      "total_time_ms": 1140000,
      "lines_added": 176,
      "lines_deleted": 45,
      "recent_tools": ["apply_patch", "codex"],
      "session_history": ["ui-build", "activity-polish"]
    }
  }
}
```

The scanner excludes `.aav` from graph nodes. Source contents are never read by history enrichment.

## Review project

Generate a clean repository with controlled commit cohorts and agent activity:

```powershell
go run ./cmd/aav-review-project -out C:\tmp\aav-extrusion-review
```

Run the desktop app, enter the generated path, and select **Load project**. Time and Work modes should produce different outward lengths while the node shell remains fixed.
