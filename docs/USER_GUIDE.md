# User guide

This guide covers the live desktop experience after a project is loaded.

## Load a project

1. Start the desktop app.
2. Enter an absolute repository path in **Project path**.
3. Select **Load project**.

The app scans the selected root, builds the deterministic graph, and then
subscribes to live focus and activity updates.

## Understand the scene

- Nodes represent the repository hierarchy.
- Directory depth determines the stable shell placement.
- Sibling angular slots are deterministic for the same repository state.
- Structural edges show the directory/file hierarchy.
- Session trail edges show the observed access path over time.

Node position stability is owned by the graph layout. Activity changes are
rendered as access points and extrusions attached to node surfaces rather than
by changing the shell radius model.

## Inspect activity

- **Access points** mark per-access anchors on the node surface.
- **Time** mode renders one outward duration extrusion per access interval.
- **Work** mode renders additions outward and deletions inward from the same
  access anchor.
- Tooltips and the inspector keep exact duration and work values even when the
  rendered geometry is visually clamped.

Unknown, binary, unsupported-encoding, empty, and pending work states stay
explicit. The renderer does not invent missing values.

## Focus and follow behavior

- **Auto-follow** keeps the camera and selection on the active file.
- Manual scene interaction places the app into an inspecting state.
- **Resume after** controls when auto-follow resumes after manual inspection.
- **Pause updates** freezes the displayed focus and trail.
- **Return live** exits a paused or replayed state and resumes live focus.

## Filter the visible graph

The left panel supports deterministic filters for:

- path or directory text
- file type
- operation
- confidence
- agent
- time range, including custom start and end

Search operates on the currently visible graph. If filters remove every visible
file, the UI shows an explicit empty state instead of fabricating a fallback.

## Review analytics

The analytics panel reports the currently visible, filter-aware values:

- total accesses and unique files
- total observed duration
- total lines added and deleted
- operation counts
- top files, directories, agents, event confidence, and work confidence

Unknown duration or work states are counted separately and remain visible in the
summary.

## Activity display controls

- Switch between **Time** and **Work** modes.
- Choose **Logarithmic** or **Linear** visual scaling.
- Set the visual cap for duration or work length.
- Toggle structure edges, labels, activity extrusions, access points, session
  trail, and render diagnostics.

Changing the visual cap changes rendered length only. Exact values remain
available in tooltips and the inspector.

## Replay and timeline

Persisted sessions can be selected from the **Session replay** panel. Replay
controls, playback, and cursor behavior are documented in
[TIMELINE_REPLAY.md](TIMELINE_REPLAY.md).

## Diagnostics

**Render diagnostics** is local-only and disabled by default. It reports CPU and
GPU timings, frame cadence, draw calls, geometry counts, and optional heap
usage when the runtime exposes those APIs. It does not send diagnostics to a
network service or a model.
