# Reliability and recovery

P9-S4 hardens local failure boundaries without placing visualization work on an
agent's critical path. Recovery remains deterministic, bounded, and local. No
recovery diagnostic contains source contents, event payloads, credentials, or a
repository path.

## Startup and persistence

1. SQLite runs `PRAGMA quick_check(1)` before migrations.
2. Each migration script and its version marker commit in one transaction. A
   failed or interrupted migration rolls back and can be retried.
3. A corrupt database is never overwritten or deleted. The database plus any
   `-wal` and `-shm` sidecars are renamed beside the original with a UTC
   `.corrupt-<timestamp>` suffix, then a clean journal is opened.
4. Health exposes fixed codes such as `database_quarantined`,
   `stale_sessions_closed`, or `session_recovery_failed`; it does not expose the
   quarantine path or an underlying SQLite payload.
5. Active or paused sessions left by a crash are marked `recovered`. Their last
   open interval closes at the earlier of observed startup and the two-minute
   idle cap, so downtime is not invented as active work.

Malformed persisted event or session rows return `store.ErrCorruptRecord` with
only the table field or numeric event sequence. Replay fails closed for that
session rather than silently presenting partial or fabricated history.

## Delivery and ordering

Local IPC validates a complete batch before submitting any event. A disconnected
collector returns within the caller's deadline; adapters remain failure-open and
do not retain an unbounded retry buffer. Because each send opens a new local
connection, the next send succeeds after collector restart without adapter
reconfiguration.

The journal ignores duplicate event IDs. Retrieval and replay use a canonical
order: session start, timestamp, optional monotonic timestamp, event ID, then
session stop. Replay also deduplicates direct fixture input. Live focus ignores
events older than its latest accepted timestamp, preventing negative intervals
or focus rollback while preserving the durable record for deterministic replay.

All normalized paths are repository-relative and slash-separated. Platform path
resolution uses Go's `filepath` semantics, including Windows volumes and Unix
roots, before the normalized protocol form is produced.

## Renderer and shutdown

A renderer subscribes to graph/focus events before calling backend `Resync()`.
The response contains the latest normalized graph snapshot and live focus only.
Graph revisions discard stale patches, and an event received during resync is
not overwritten by an older focus response.

Desktop shutdown is idempotent and ordered: close IPC, drain accepted ingress,
cancel bounded diff work, flush the journal queue, then close SQLite. Collector
failure or a saturated queue may lose visualization history, but never changes
agent output, exit status, working files, or repository contents.

## Reproduction

From the repository root:

```powershell
go test ./internal/store ./internal/session ./internal/replay ./internal/ingest ./internal/ipc ./adapter/go
go test ./...
go vet ./...
```

From `apps/desktop`:

```powershell
go test ./...
go vet ./...
```

From `apps/desktop/frontend`:

```powershell
npm ci
npm run format:check
npm run lint
npm run typecheck
npm test -- --run
npm run build
```
