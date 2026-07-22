# ADR 0013: Rendering level of detail

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Large repositories and long sessions can produce tens of thousands of repeated nodes, markers, edges, and labels.

## Decision
Use instanced meshes for nodes/points, buffer lines for edges/extrusions, distance and count-based label LOD, and aggregated dense-history markers.

## Alternatives
One React component and mesh per item is simpler but scales poorly in draw calls and reconciliation.

## Consequences
Geometry updates require explicit buffer lifecycle and numerical tests.

## Evidence
Repeated primitives share materials and geometry, making instancing the natural GPU representation.
