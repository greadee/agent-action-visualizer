# Architecture

## Goals and constraints

Agent Action Visualizer is a local-first observability system. It converts deterministic development events into a stable repository graph, session state, and replayable activity geometry. It does not call an AI API, change an agent prompt, or require model-visible reporting.

The system is split so a renderer, adapter, persistence engine, or transport can change independently. Every event carries provenance and confidence. Adapters fail open: collector failure can lose visualization data, but cannot alter an agent decision, output, exit code, or working file.

## System boundaries

```text
agent/editor/fs/git
       |
       v
adapter -> local authenticated transport -> bounded ingress queue
                                            |
                                            v
                         validate -> normalize -> deduplicate
                                            |
                  +-------------------------+-----------------------+
                  |                         |                       |
                  v                         v                       v
             session engine          graph service          async diff workers
                  |                         |                       |
                  +-------------------------+-----------------------+
                                            |
                                            v
                                          SQLite
                                            |
                              snapshot + incremental event stream
                                            |
                                            v
                           Wails bridge -> React/R3F renderer
```

## Event lifecycle

1. An adapter receives a supported native event or observes a local change.
2. It performs bounded parsing, adds adapter metadata, submits a versioned envelope, and exits successfully without output.
3. Local IPC uses owner-scoped OS access control, size-limits the envelope, and validates a complete batch before offering any event to a bounded priority queue.
4. The normalizer resolves the project root, rejects lexical and symbolic-link escapes, minimizes metadata, and produces canonical relative paths.
5. Deduplication and pre/post correlation enrich the event without rewriting raw evidence.
6. The session engine applies active-file priority, closes/open access intervals, and caps idle time.
7. Graph changes and line-delta work are scheduled asynchronously.
8. One storage worker persists ordered state in SQLite; raw source contents are not stored by default.
9. The publisher emits sanitized graph patches and session updates through Wails events.
10. The renderer consumes normalized state only; it never parses raw tool or terminal output.

## Core packages

- `protocol/`: versioned JSON Schema plus Go and TypeScript contract types.
- `adapter/go`: public adapter descriptor, capability, bounded emitter, path, and test contracts.
- `adapter/go/wrapper`: transparent process lifecycle runner, structured-stream parser seam, and fallback observer seam.
- `internal/ingest`: validation, bounded queueing, deduplication, and normalization.
- `internal/session`: lifecycle, focus priority, access intervals, idle handling, and replay state.
- `internal/project`: safe scanning and ignore handling.
- `internal/graph`: stable identities, hierarchy, deterministic layout, snapshots, and patches.
- `internal/diff`: asynchronous structured-patch, snapshot, and Git numstat delta sources.
- `internal/store`: migrations and repositories over pure-Go SQLite.
- `internal/ipc`: local transport, authentication, payload limits, and frontend publication.
- `internal/adapters`: Codex, filesystem/Git, and optional integration implementations.
- `apps/desktop`: Wails lifecycle and a React/TypeScript/Three.js presentation layer.

## Active-file resolution

Candidates are ordered by evidence, operation, and recency: explicit native file events; structured tool events; correlated command/filesystem events; recent high-confidence writes; then recent observations. Writes, patches, creates, moves, and deletes outrank reads at equal evidence. A move preserves identity; a delete uses a temporary tombstone. Focus changes close the prior access interval and retain the previous node.

## Failure and overload behavior

- Hook adapters use short local deadlines, emit no stdout/stderr on ordinary failure, and always exit zero.
- The ingress queue is bounded. It coalesces duplicate reads first, then drops low-confidence/read activity before create/write/move/delete/session events.
- SQLite and diff work occur off the adapter critical path.
- Invalid versions, oversized payloads, path traversal, and out-of-root paths are rejected and diagnosed locally.
- Renderer disconnects do not stop collection. Reconnection requests a fresh snapshot followed by patches.
- Recovery closes stale open intervals at the last trustworthy timestamp and marks them recovered.

## Data retention and privacy

The default database contains relative paths, allowlisted normalized metadata, timestamps, counts, hashes where needed, and single-line redacted command/tool labels. Raw command fields are discarded before persistence. Unsupported metadata keys are dropped at normalized ingress and again at the SQLite boundary, so source contents, environment variables, prompts, model responses, credentials, and telemetry are not retained by default. Temporary before/after snapshots are opt-in, constrained to the selected root, and deleted after delta calculation.

## Renderer model

Directory depth maps to stable shells. Hash-derived angular slots keep sibling positions stable across rescans. Instanced nodes and activity points minimize draw calls; structural and session-trail edges use buffer geometry. Activity anchors follow a deterministic Fibonacci distribution. Each time interval extends outward from its access anchor; completed intervals use their normalized session duration, while an active interval grows only to the same idle cap used by the session engine. Time lengths retain exact millisecond values for inspection while visual lengths use bounded linear or logarithmic scaling. Work-delta workers prefer structured counts, use explicitly allowed snapshot correlation, batch Git numstat fallback, and retain unknown/binary/encoding states. Additions project outward from an access anchor; deletions from that same anchor project inward. Camera focus uses quaternion interpolation without relayout and yields to manual controls.
