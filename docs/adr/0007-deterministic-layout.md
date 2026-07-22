# ADR 0007: Deterministic hierarchical spherical layout

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Users need spatial memory across rescans and focus changes.

## Decision
Map hierarchy depth to shells and hash-stable sibling angular slots. Preserve stored positions and place new nodes without global force simulation.

## Alternatives
A continuous force layout is visually fluid but rearranges the project and consumes ongoing CPU.

## Consequences
Hierarchy remains readable and deterministic; perfect collision minimization is traded for stability.

## Evidence
Stable hash order and radial shells are reproducible and testable across platforms.
