# ADR 0009: No model calls or model-visible reporting

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Observability must not consume model tokens, affect reasoning, or expose visualization state to the agent.

## Decision
Do not call AI APIs, modify prompts, add reporting tools, parse generated summaries, or emit hook output. Successful adapters exit zero with empty stdout/stderr.

## Alternatives
Agent narration is easier to interpret but changes cost, latency, and behavior.

## Consequences
Some unsupported-agent reads remain unknowable and must be labeled inferred.

## Evidence
Deterministic hook, process, filesystem, and Git sources cover useful activity without model execution.
