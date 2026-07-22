# ADR 0006: SQLite persistence

- Status: Accepted
- Date: 2026-07-21
- Supersedes: none
- Superseded by: none

## Context
Projects, events, intervals, graph state, settings, diagnostics, and replay require ordered durable local storage.

## Decision
Use SQLite in WAL mode through a pure-Go driver, owned by one migration and repository layer.

## Alternatives
Flat files complicate querying and atomic recovery. A server database violates local-first packaging.

## Consequences
The system gains transactions and indexed replay while requiring migration and corruption handling.

## Evidence
SQLite is embedded, cross-platform, and suited to append-heavy session metadata.
