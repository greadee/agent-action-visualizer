# Generic command wrapper

`aav-wrapper` observes the lifecycle of any local command without requiring an
agent-specific hook. It is deterministic, local, and metadata-only. It does not
call a model, change a prompt, or report visualization state to the wrapped
process.

## Build and run

From the repository root:

```powershell
go build -o .cache/aav-wrapper.exe ./cmd/aav-wrapper
.\.cache\aav-wrapper.exe --project-root . -- agent-command argument
```

On macOS or Linux:

```sh
go build -o .cache/aav-wrapper ./cmd/aav-wrapper
./.cache/aav-wrapper --project-root . -- agent-command argument
```

Use `--` before the command so command flags are not interpreted as wrapper
flags. Wrapper flags are:

- `--project-root`: project associated with lifecycle events; defaults to the
  current working directory.
- `--session-id`: stable caller-provided session ID; a local random ID is
  generated when omitted.
- `--agent-type`: metadata label; defaults to `generic`.
- `--endpoint`: local collector endpoint; defaults to
  `AAV_COLLECTOR_ENDPOINT` or the platform's per-user endpoint.
- `--filesystem-fallback`: enables recursive metadata-only repository
  observation; defaults to `true`. Set it to `false` when an embedding supplies
  complete native or structured file evidence.

## Preservation contract

The child receives the exact argument vector and inherited environment. The
wrapper does not add, remove, or rewrite child environment variables. Stdin is
connected directly. Stdout and stderr remain separate and preserve every byte,
including for large streams and nonzero exits.

The wrapper returns the child's native exit code. On Unix, the child runs in a
new process group; `SIGINT`, `SIGTERM`, `SIGHUP`, and `SIGQUIT` are forwarded to
that group, and the wrapper re-raises a terminating child signal so callers see
the same signal. On Windows, wrapper and child retain the shared console
control-event path and the wrapper returns the child's native process code.

Collector disconnects, malformed parser output, parser panic, parser timeout,
fallback failure, and queue saturation can lose visualization evidence only.
They do not alter child I/O, files, or exit behavior.

## Lifecycle evidence

The wrapper emits exact `wrapper` source events in this order:

1. `session_started`
2. `command_started`
3. zero or more structured-stream or filesystem fallback events
4. `command_completed`
5. `session_stopped`

Lifecycle events store the executable basename, process ID, project root,
timestamps, duration, status, exit code, and terminating signal where
applicable. They never store command arguments, environment values, stdin,
stdout, or stderr.

Delivery uses a 64-batch queue and a 75-millisecond local send deadline by
default. Parsing uses a separate 64-chunk queue with 32 KiB chunks. Queue
overflow marks the next accepted chunk, or EOF, with `DroppedBefore`; parsers
must discard partial framing at that boundary. The wrapper waits at most 25
milliseconds for parser drain after the child exits.

## Integration interfaces

`adapter/go/wrapper.StructuredStreamParser` receives bounded copies of output
only when explicitly configured by an embedding integration. The original
output destination is written first, so parsing cannot suppress or change
bytes. Parsers must accept arbitrary chunk boundaries and must emit complete,
validated protocol events with honest source and confidence values.

`adapter/go/wrapper.FallbackObserver` receives session, root, agent, and process
metadata after the child starts, runs asynchronously, and is canceled when the
child exits. The standalone CLI installs the P7-S3 implementation by default.

The fallback recursively watches non-ignored directories without reading or
persisting source contents. It shares `.gitignore`, `.aavignore`, and default
dependency/build exclusions with the project scanner. Bursts are debounced and
coalesced per path through bounded queues. Each ready batch uses one Git status
inspection and one Git numstat inspection for supported tracked files.

Create, modify, and delete facts use `filesystem/observed`. Rename and move
facts are emitted only when Git evidence or a unique metadata match correlates
the old and new path; those events use `filesystem/correlated`. Ambiguous
renames remain separate delete/create events. Binary and unknown Git evidence
remain explicit rather than receiving invented line counts.

Filesystem events wait briefly at a bounded evidence gate. A matching
native-hook or structured-stream event suppresses the fallback duplicate;
stronger evidence is never delayed. Unmatched fallback evidence is emitted
after 100 milliseconds or flushed during bounded shutdown.

Agent-specific structured parsers remain opt-in compile-time integrations. The
wrapper never parses ordinary terminal output automatically.

## Validation

Contract tests cover exact arguments and environment values, stdin, independent
stdout/stderr, success, exit code 23, 2 MiB output on each stream, cancellation,
Unix signal forwarding and re-raising, parser saturation, blocked observers,
disconnected collectors, metadata redaction, cross-source duplicate
suppression, burst overload, long sessions, recursive ignore behavior, rename
correlation, real filesystem notification, and executable-level behavior.
