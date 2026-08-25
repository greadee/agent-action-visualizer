# Privacy and no-model-call boundary

Agent Action Visualizer is local-first observation software. The intended
privacy boundary is that repository activity stays on the local machine unless a
user explicitly exports or shares something outside the application.

## What the application does not do

- It does not call an AI model to observe activity.
- It does not modify agent prompts.
- It does not inject visualization summaries back into agent context.
- It does not require public network listeners.
- It does not persist raw source contents, prompts, model responses, command
  arguments, environment variables, or transcript contents by default.

This boundary is architectural, not merely UI policy. See
[ARCHITECTURE.md](ARCHITECTURE.md), [SECURITY.md](SECURITY.md), and
[ADR 0009](adr/0009-no-model-calls.md).

## What is retained locally

The default local journal stores only the normalized data needed for replay and
inspection:

- session and event identity
- timestamps
- project-relative paths
- event type, source, and confidence
- deterministic counts such as duration and line deltas
- bounded allowlisted metadata used by reducers and inspectors

The selected local project root is retained so the desktop app can reopen a
project. That path remains local to the machine.

## What adapters are allowed to inspect

Adapters may inspect bounded local evidence needed to produce normalized
metadata:

- Codex hooks may parse documented hook JSON in memory.
- Work-delta calculation may inspect structured patches, allowed before/after
  snapshots, or Git numstat output in memory.
- The generic wrapper may forward child stdout and stderr unchanged while an
  optional parser inspects bounded copies when explicitly configured.

Those paths exist to derive local metadata only. They do not authorize source
retention or model-visible reporting.

## Evidence that observation causes no model calls

Several repository contracts and tests exist specifically to prove the boundary:

- [docs/CODEX_HOOK.md](CODEX_HOOK.md): the hook writes no stdout/stderr on
  ordinary paths, exits zero on ordinary failure, and treats collector failure
  as local success.
- [docs/CODEX_E2E.md](CODEX_E2E.md): the reproducible hook-path fixture does not
  invoke Codex model execution and validates that the hook remains silent.
- [docs/GENERIC_WRAPPER.md](GENERIC_WRAPPER.md): wrapper observation preserves
  child I/O and exit behavior and does not parse ordinary terminal output by
  default.
- [docs/SECURITY.md](SECURITY.md): threat controls explicitly cover prompt,
  response, command, and environment non-retention.

## Residual privacy boundaries

- Same-user processes are inside the current IPC trust boundary.
- Secret redaction is defense-in-depth rather than a complete secret detector.
- Optional exported build artifacts or screenshots are outside the local-only
  boundary and should be handled intentionally.
