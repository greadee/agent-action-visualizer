# ADR 0008: Native hooks before filesystem inference

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Filesystem watchers observe writes but generally cannot prove reads or the agent's current target.

## Decision
Prefer supported native lifecycle/tool events, then structured streams, explicit editor integrations, filesystem/Git observation, and finally heuristics.

## Alternatives
Filesystem-only collection is broadly compatible but cannot claim exact focus.

## Consequences
Adapters expose capabilities and the UI must communicate fallback limitations.

## Evidence
Current Codex hooks provide SessionStart, PreToolUse, PostToolUse, SubagentStart/Stop, and Stop payloads with structured tool input.
