# ADR 0012: Private, metadata-only retention

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Development events can contain source, prompts, command secrets, environment values, and absolute paths.

## Decision
Persist repository-relative paths and redacted metadata only by default. Do not retain source, prompts, responses, environment variables, or telemetry. Diagnostic snapshots are explicit and temporary.

## Alternatives
Full transcripts simplify later interpretation but create unacceptable privacy and secret-retention risks.

## Consequences
Some retrospective calculations are unavailable unless captured safely at event time.

## Evidence
Required analytics depend on counts and timestamps rather than source contents.
