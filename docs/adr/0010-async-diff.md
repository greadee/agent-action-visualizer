# ADR 0010: Asynchronous diff calculation

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Line deltas may require parsing patches, comparing temporary snapshots, or batching Git diff operations.

## Decision
Hooks enqueue evidence only. Bounded workers calculate deltas later, preferring structured patch counts and recording unknown rather than inventing values.

## Alternatives
Synchronous hook diffing improves immediate completeness but blocks the agent critical path.

## Consequences
Work-mode values may arrive after focus events and update incrementally.

## Evidence
Diff cost grows with file/repository size and does not belong in the submission latency budget.
