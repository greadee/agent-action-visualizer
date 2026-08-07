# ADR 0012: Private, metadata-only retention

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Development events can contain source, prompts, command secrets, environment values, and absolute paths.

## Decision
Persist repository-relative paths and allowlisted, redacted metadata only by default. Discard raw command fields, source, prompts, responses, environment variables, credentials, and telemetry at normalized ingress and again at the storage boundary. Diagnostic snapshots are explicit and temporary.

## Alternatives
Full transcripts simplify later interpretation but create unacceptable privacy and secret-retention risks.

## Consequences
Some retrospective calculations are unavailable unless captured safely at event time.

## Evidence
Required analytics depend on counts and timestamps rather than source contents. Storage regression tests inject a unique source/secret marker and verify that it is absent from persisted event JSON while replay metadata survives.
