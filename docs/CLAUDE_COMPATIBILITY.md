# Claude Code compatibility evidence

This record captures the Claude Code integration surface reviewed for P7-S4 on
2026-08-04. It defines an optional, unverified compatibility boundary. It is
not an installation guide, adapter implementation, or permission to modify
Claude Code settings.

## Local environment evidence

- No `claude` executable was discoverable on `PATH`.
- No project `.claude` directory was present in this checkout.
- No user-level Claude settings directory was found at the conventional
  Windows locations inspected for this slice.
- The global npm package root did not contain `@anthropic-ai/claude-code`.
- No `CLAUDE_*` or `ANTHROPIC_*` environment-variable keys were present. Values
  were not read or recorded.
- WSL is not installed on this host. Anthropic documents Windows support via
  WSL or Git Bash, so no local Claude version, hook execution, settings merge,
  stdout/stderr behavior, or exit-code behavior could be verified here.

## Official interface reviewed

Anthropic's current [hooks reference](https://code.claude.com/docs/en/hooks)
documents command hooks that receive JSON on standard input. The stable
candidate lifecycle events are `SessionStart`, `SessionEnd`, `PreToolUse`, and
`PostToolUse`; tool events include `tool_name`, `tool_input`, and a tool-use
identifier. `PostToolUse` is the strongest candidate for completed local file
tool evidence. The same reference documents command-handler configuration in
user settings, project settings, local project settings, and managed policy
settings.

The reviewed settings locations are:

- `~/.claude/settings.json` for user-local hooks.
- `.claude/settings.json` for shareable project hooks.
- `.claude/settings.local.json` for local project hooks.

Anthropic documents that hooks can return decisions or context through stdout
and exit-code handling. AAV must never use those control paths: a future
observer must be silent on stdout and stderr, return zero for ordinary
failures, enforce a short local deadline, and treat collector failure as local
success. It must never inspect transcripts, retain prompts, source content,
tool responses, command text, credentials, or model output.

## Compatibility boundary

| Semantic | Status | Handling |
| --- | --- | --- |
| Command-hook lifecycle | officially documented; locally unverified | Do not claim runtime compatibility until a local `claude` version is available. |
| Session start/end | candidate | Map only if validated against a disposable trusted project. |
| File read/write/edit paths | candidate | Extract only validated, root-confined paths from supported tool input; discard content fields. |
| Tool success/failure | candidate | `PostToolUse` may prove successful completion; unsupported or absent hooks remain unknown. |
| Exact duration | unverified | Do not infer duration from hook receipt times without an ordering study. |
| Exact work delta | unsupported by hooks alone | Use the existing bounded diff pipeline only after a normalized write event; otherwise report unknown. |
| Installation and settings mutation | intentionally unimplemented | No project, user, managed, plugin, or skill hook configuration was changed. |
| Existing-session attachment | unsupported by this slice | No passive interface was reviewed or assumed. |

## Sanitized contract fixtures

[`testdata/claude`](../testdata/claude) contains schema-shaped examples of the
reviewed command-hook envelopes. They are documentation fixtures, not captured
sessions, and contain placeholders rather than source contents, prompts,
commands, transcripts, credentials, or real paths. They do not establish that
any uninstalled Claude Code version emits every optional field.

## Required future verification

Before optional adapter implementation is authorized, validate the installed
`claude` version and `claude doctor` output, inspect the active settings scope,
and exercise a disposable trusted project. The validation must prove silent
failure-open behavior, path-root enforcement, malformed payload handling,
collector disconnect handling, hook timeout behavior, exact stdout/stderr
preservation, and unchanged Claude exit behavior. No model calls, prompt
changes, or model-visible visualization reporting may be introduced.
