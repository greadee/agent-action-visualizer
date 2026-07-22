# ADR 0014: Phase/slice Git workflow

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
The build spans platform, storage, protocol, rendering, adapters, security, and packaging; recoverability matters.

## Decision
Implement one coherent P#-S# slice per validated commit, update the implementation log, and push before starting another slice. Work occurs on `codex/initial-build` after a minimal `main` bootstrap.

## Alternatives
One large feature branch commit obscures failures; many arbitrary checkpoints lack semantic recovery points.

## Consequences
History is reviewable and remote-backed, with small bookkeeping overhead.

## Evidence
Phase/slice boundaries correspond to independently testable architectural outcomes.
