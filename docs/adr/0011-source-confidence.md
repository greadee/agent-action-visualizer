# ADR 0011: Source-confidence model

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Native payloads, correlations, filesystem changes, and heuristics provide different certainty.

## Decision
Every normalized event records source type and one of `exact`, `correlated`, `observed`, or `inferred`. UI and filters expose both.

## Alternatives
A single activity stream looks simpler but misrepresents inferred focus as fact.

## Consequences
Consumers can reason about trust while adapters must justify classification.

## Evidence
The same write can be reported by a native tool hook and filesystem watcher with different semantics.
