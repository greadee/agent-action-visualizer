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

Migrations execute and record their version in one transaction, so an interrupted
or failed script can be retried without retaining a partial schema. Startup runs a
bounded SQLite integrity check. A corrupt database and its WAL/SHM sidecars are
renamed to a timestamped quarantine path before a clean journal is created; the
application never deletes the quarantined evidence automatically.

## Evidence
SQLite is embedded, cross-platform, and suited to append-heavy session metadata.
P9-S4 tests failed-migration rollback and retry, corrupt-file quarantine and byte
preservation, malformed-row diagnostics, reopen, and stale-session recovery.
