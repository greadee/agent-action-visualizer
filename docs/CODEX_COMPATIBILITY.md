# Codex compatibility evidence

This record captures the Codex integration surface reviewed for P6-S1 on
2026-07-27. It is evidence for a future local adapter, not an installation
guide and not an adapter implementation.

## Evidence reviewed

- Installed executable: `C:\Program Files\WindowsApps\OpenAI.Codex_26.721.4979.0_x64__2p2nqsd0c76g0\app\resources\codex.exe`.
- The package-directory version is `26.721.4979.0` for Windows x64. Direct
  `codex --version` and `codex app-server --help` probes returned `Access is
  denied` in this managed environment, so this is package evidence rather
  than a verified CLI semantic version.
- The official Codex manual was fetched on the review date. The relevant
  primary-source sections are [Hooks](https://learn.chatgpt.com/docs/hooks),
  [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference.md),
  and [App Server](https://learn.chatgpt.com/docs/app-server).
- This checkout has no `.codex/config.toml` or `.codex/hooks.json`. A
  user-level config exists, but no user-level `hooks.json` was present. Its
  values were not inspected or recorded; a key-only review found no inline
  hook configuration.

## Supported lifecycle surface

Codex documents command lifecycle hooks, enabled by default through
`features.hooks` (with `features.codex_hooks` retained as a deprecated alias).
Each command hook receives one JSON object on standard input. Common fields
are `session_id`, `transcript_path`, `cwd`, `hook_event_name`, and `model`.
Turn-scoped hooks add `turn_id`; tool hooks provide the canonical tool name and
tool-specific JSON input. `PostToolUse` also provides a tool response.

The documented event names are `SessionStart`, `SessionEnd`, `PreToolUse`,
`PermissionRequest`, `PostToolUse`, `PreCompact`, `PostCompact`,
`UserPromptSubmit`, `SubagentStart`, `SubagentStop`, and `Stop`. `PreToolUse`
and `PostToolUse` cover supported local tools including Bash, `apply_patch`,
MCP tools, and other local function tools. They do not cover hosted tools such
as WebSearch, and specialized paths can opt out of the normal hook path.

Only `type: "command"` handlers run in the reviewed interface. `prompt` and
`agent` handlers are parsed but skipped. The documented Windows command
override is `commandWindows`; inline TOML also accepts `command_windows`.
Codex starts matching command hooks concurrently in the session working
directory and requires trust review for non-managed command hooks.

### Safe adapter boundary

Hooks can affect the agent when they return output or particular exit codes:
their output may become developer context, warnings, or substituted tool
feedback. An observer for this repository must therefore use a short local
deadline and, on all ordinary paths, write nothing to stdout or stderr and
exit `0`. It must discard tool input/output contents after extracting allowed
metadata, never read the transcript, and treat collector failures as local
success. This preserves Codex output, working files, and exit behavior.

`PostToolUse` is the strongest documented candidate for the future adapter
because it runs after a supported local tool completes, including non-zero
Bash exits. It remains incomplete coverage, so P6-S2 must mark absent or
unsupported evidence as unknown rather than infer an access or edit.

## Configuration and installation surfaces

Codex discovers both `hooks.json` and inline `[hooks]` tables next to active
configuration layers. The useful supported locations are:

- User level: `~/.codex/hooks.json` or `~/.codex/config.toml`.
- Trusted project level: `<project>/.codex/hooks.json` or
  `<project>/.codex/config.toml`.

If a layer contains both forms, Codex merges them and warns at startup; one
representation per layer is preferred. Project-local hooks are not loaded for
an untrusted project. Managed and plugin hook sources have separate policy and
trust behavior and are out of scope for the project-local P6 adapter.

P6-S3 must use these exact surfaces, preserve unrelated configuration, back up
before writes, and make installation idempotent. P6-S1 did not create, change,
or trust any Codex hook configuration.

## App Server and structured events

`codex app-server` is a documented bidirectional JSON-RPC 2.0 interface. Its
default `stdio` transport uses newline-delimited JSON; WebSocket is documented
as experimental and unsupported. A client must send `initialize`, then
`initialized`, before any other request. The server can emit thread, turn, and
item notifications, including `item/started` and `item/completed`.

This is a client-hosting interface, not a documented passive attachment point
for an already-running desktop session. Starting a server, a thread, or a turn
would be an active Codex integration and is not suitable for the P6-S2
lightweight observer. In particular, `thread/inject_items` is model-visible
and `thread/shellCommand` runs outside the thread sandbox; neither may be used
by this visualizer.

The CLI documentation also describes a JSONL mode for `codex exec`, but it is
a process invocation surface, not evidence that the desktop session can be
observed without wrapping it. P7 owns the generic wrapper path.

## Status by semantic

| Semantic | Status | Evidence and handling |
| --- | --- | --- |
| Session start/end | supported | Hook events expose a session ID and working directory. Session-end output is advisory. |
| Local tool access | supported, partial | `PreToolUse`/`PostToolUse` cover documented local tool paths only. Unsupported paths stay unknown. |
| File path and operation | conditional | Tool input shape is tool-specific. Extract only validated paths and operation metadata; do not retain commands, patches, or responses. |
| Tool completion | supported, partial | `PostToolUse` follows supported tools, including non-zero Bash exits. It cannot undo prior side effects. |
| Exact duration | not yet verified | Hook timestamps can bracket local observations, but P6-S2 must validate ordering and idle handling before claiming exact duration. |
| Exact work delta | unsupported by hooks alone | Requires structured patch evidence or the existing bounded diff pipeline; otherwise report unknown. |
| Existing-session attachment | unsupported | No reviewed supported passive App Server attachment interface was found. |
| Hook trust and configured runtime | unverified locally | This environment cannot execute the installed binary and has no configured project hook. P6-S2/P6-S4 must test a disposable trusted repository. |

## Sanitized fixtures

The fixtures in [`testdata/codex`](../testdata/codex) contain no source
contents, commands, credentials, transcript data, or real paths. They model
only documented envelope fields and deliberately replace content-bearing
values with `<redacted>`. They are contract fixtures for the future adapter,
not captured session data and not a claim that every optional field is emitted
by every Codex release.

## Deferred verification

- Verify the exact CLI build and schema generation commands in an environment
  permitted to execute the installed binary.
- Exercise trusted project-local and user-level installation, failure-open
  behavior, timeouts, output preservation, and exit-code preservation in P6-S2
  through P6-S4.
- Determine supported path extraction per observed tool payload without
  retaining command, patch, response, or transcript contents.
