# Codex lifecycle hook adapter

P6-S2 provides `aav-codex-hook`, a silent command-hook executable that reads
one documented Codex hook JSON object from standard input and submits a
metadata-only event batch to the local Agent Action Visualizer collector.

## Critical-path behavior

- Input is capped at 1 MiB and decoded once.
- At most 128 normalized events are produced for one hook invocation.
- Local submission has a 75 ms total deadline.
- The wire frame is capped at 256 KiB and the collector accepts at most 32
  concurrent connections.
- Parsing, translation, queue pressure, timeout, malformed payload, and a
  disconnected collector all return silently with exit code zero.
- The hook never writes stdout or stderr and never returns hook output to
  Codex. It therefore cannot add model context, replace a tool result, or
  change Codex approval behavior.

The desktop receiver immediately offers validated batches to the existing
256-event bounded ingestion queue. Deterministic event IDs suppress duplicate
deliveries within the ingress window. Rendering and work-delta calculation
remain off the hook path.

## Local transport

Windows uses a per-user named pipe with an owner/system ACL. macOS and Linux
use a user-config-directory Unix socket with mode `0600`. The executable and
desktop derive the same endpoint; `AAV_COLLECTOR_ENDPOINT` can override it for
isolated tests and diagnostics. No public or loopback network listener is
created.

## Translation contract

| Codex hook | Normalized evidence |
| --- | --- |
| `SessionStart` | `session_started`, except `source: compact`, which does not create a new session |
| `SessionEnd` | `session_stopped` |
| `SubagentStart` / `SubagentStop` | `agent_started` / `agent_stopped` |
| `PreToolUse` | `tool_started` with `tool_use_id` correlation |
| `PostToolUse` | `tool_completed` plus recognized structured file evidence |

`apply_patch` headers provide structured create, patch, delete, rename, and
move paths plus addition/deletion counts. Recognized file MCP/local tools use
only explicit path fields. Ambiguous or unsupported tools produce no invented
file event. Tool-derived file semantics use `correlated` confidence because a
post-hook proves the tool produced output but does not provide a stable,
tool-independent success field.

Commands, patch text, tool responses, model names, permission modes,
transcript paths, prompts, environment variables, and source contents are not
copied into normalized events. Absolute paths can exist only in the local wire
frame long enough for the collector to normalize them against the selected
project root; escaping paths are rejected by ingress.

## Verification

Run the repeatable hook-path benchmark with:

```powershell
go test ./internal/adapters/codex -run '^$' -bench BenchmarkRunHook -benchmem -count 5
go test ./internal/ipc -run '^$' -bench BenchmarkLocalRoundTrip -benchmem -count 5
go test ./cmd/aav-codex-hook -run '^$' -bench BenchmarkHookProcessRoundTrip -benchtime=20x -count 5
```

On the P6-S2 Windows validation host (AMD Ryzen 5 9600X, Go 1.26.3,
windows/amd64), the five-run medians were 12.575 microseconds for bounded hook
decode/translation/no-op submission and 73.212 microseconds for a complete
per-user named-pipe round trip. The process-level benchmark includes executable
startup and measured a 6.890 ms five-run median (20 launches per run). It is
the closest reproducible hook-overhead measurement without
installing the hook into Codex; P6-S4 owns the real Codex end-to-end result.

P6-S3 provides project/user install, dry-run, status, isolated test, and
reversible uninstall behavior. See [the Codex installer guide](CODEX_INSTALLER.md).
P6-S4 adds reproducible session evidence, restart/reconnect coverage, and
fresh hook-path measurements. See [the end-to-end validation record](CODEX_E2E.md).
