# Codex reproducible end-to-end validation

P6-S4 validates the supported local hook protocol against a disposable Git
repository without starting a model turn, reading a transcript, or changing a
user-level Codex configuration. The fixture invokes the hook process with the
documented JSON shapes captured in P6-S1, including the same local IPC path
used by a trusted Codex project hook.

Run the deterministic validation:

```powershell
go test ./cmd/aav-codex-hook -run '^TestReproducibleCodexFixtureSession$' -count 1 -v
go test ./apps/desktop -run '^TestNativeMoveAndDeletePreserveGraphIdentity$' -count 1 -v
```

The first command creates a temporary Git repository and database outside the
checkout, then exercises the installed hook process entry point. No source
snapshot, prompt, model response, transcript, or credential is stored.

| Scenario | Verified local evidence |
| --- | --- |
| Session lifecycle | `SessionStart` starts and `SessionEnd` closes the session and every access interval. |
| Read/create/patch | Documented `PostToolUse` payloads produce normalized focus and structured `+2 / -1` work evidence. |
| Move/delete | The fixture emits move and delete evidence; the desktop integration test retains the stable move ID and exposes a tombstone after refresh. |
| Focus and time | Ordered access intervals preserve active and previous paths and close with non-negative, idle-capped durations. |
| Disconnect/reconnect | A hook call against a stopped collector exits zero with no output and no session mutation; the next hook after local collector restart is accepted. |
| Persistence/replay | Sanitized normalized records reopen from SQLite and reconstruct the same access trail with the pure session engine. |
| Model isolation | The hook writes no stdout/stderr and the fixture never invokes Codex model execution. |

## Hook-path measurements

Commands:

```powershell
go test ./internal/adapters/codex -run '^$' -bench BenchmarkRunHook -benchmem -count 5
go test ./internal/ipc -run '^$' -bench BenchmarkLocalRoundTrip -benchmem -count 5
go test ./cmd/aav-codex-hook -run '^$' -bench BenchmarkHookProcessRoundTrip -benchtime=20x -count 5
```

On the P6-S4 Windows/amd64 host (AMD Ryzen 5 9600X, Go 1.26.3), five-run
medians were 11.866 microseconds for decode/translation/no-op send, 51.933
microseconds for a named-pipe round trip, and 7.553 milliseconds for the
process-level hook invocation (20 launches per run).

## Live Codex boundary

The installed AppX Codex package is `26.721.4979.0`. Both restricted and
elevated `codex --version` and `codex --help` probes returned Windows
`Access is denied`, so this host cannot start `codex exec`, use `/hooks` for
trust review, or produce a real authenticated agent turn from the terminal.

The official manual confirms that project hooks need trust review and that
`codex exec` is the supported non-interactive path. A real-session check still
requires a host where the Codex CLI can execute and where the disposable
project hook can be reviewed or intentionally invoked with
`--dangerously-bypass-hook-trust`. That future run must keep visualization
output empty and must not send visualization state to the model.
