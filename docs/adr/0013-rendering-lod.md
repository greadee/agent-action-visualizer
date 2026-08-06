# ADR 0013: Rendering level of detail

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context

Large repositories and long sessions can produce tens of thousands of repeated nodes, markers, edges, and labels.

## Decision

Use instanced meshes for nodes/points, buffer lines for edges/extrusions, distance and count-based label LOD, and aggregated dense-history markers.

P9-S1 fixes the first production thresholds at 256 detailed activity accesses
and 2,048 rendered access-point instances. Dense activity selection always
retains the inspected access, active intervals, the maximum duration/addition/
deletion evidence used by visual scaling, the newest 75 percent of the detail
budget, and deterministic samples from older history. Selected labels remain
visible; hover labels reduce their distance threshold above 1,000 and 5,000
nodes. Node icosahedron detail steps down at the same count thresholds.

The WebGL renderer exposes optional local diagnostics for draw calls,
triangles, lines, geometry/texture allocations, CPU submission p95, frame rate,
GPU timer-query p95 where supported, and JavaScript heap use where exposed by
the runtime. Diagnostics never leave the renderer or enter agent context.

P9-S2 fixes the reproducible scale campaign at 100, 1,000, 5,000, 10,000,
and 20,000 nodes, with history fixtures bounded at 100,000 exact accesses.
Development-only URL profiles feed the normal graph/focus inputs and the normal
browser preview bridge; production behavior does not synthesize fixture data.
Structure-edge positions are written directly into one fixed-size typed buffer
to avoid temporary nested arrays that otherwise amplify peak allocation at
large node counts.

## Alternatives

One React component and mesh per item is simpler but scales poorly in draw calls and reconciliation.

## Consequences

Geometry updates require explicit buffer lifecycle and numerical tests.
Inspected dense-history evidence is promoted into the bounded detail set, while
the complete exact record remains available to replay, analytics, and the
inspector. WebGL line buffers are disposed whenever normalized input changes.
At 20,000 unfiltered nodes, the shell is deliberately treated as an overview;
search, filtering, focus, and the selected label provide individual
inspectability instead of retaining expensive per-node labels or geometry.

## Evidence

Repeated primitives share materials and geometry, making instancing the natural GPU representation.
