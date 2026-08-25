# Known limitations

This file records the current release boundary as of August 17, 2026.

## Platform and packaging

- Windows desktop development and packaging are validated locally.
- Linux and macOS packaging are validated in CI only, not on this host.
- Release artifacts are unsigned.
- macOS artifacts are not notarized.
- NSIS installer generation is validated in CI rather than on this host because
  NSIS is not installed locally.

## Codex boundary

- The documented Codex hook protocol, installer, failure-open behavior, and
  silent observation contract are validated through reproducible local fixtures
  and tests.
- A real authenticated Codex turn cannot be launched from this Windows host
  because the installed AppX Codex CLI still returns `Access is denied` when
  invoked directly.
- Project-scope Codex hooks still depend on Codex trust review and local
  runtime support outside this repository.

## Replay and analytics

- Replay history begins only after this build has persisted sessions for the
  selected project.
- Replay depends on the local journal remaining available.
- Analytics reflect the currently visible, filtered graph and trail only.
- Subagent activity is only as specific as the adapter-provided agent ID.

## Security and recovery boundary

- Same-user processes remain inside the current local IPC trust boundary.
- Secret redaction is defense-in-depth rather than a universal detector.
- Release validation refreshes npm and Go vulnerability data; future releases
  must repeat those network-backed checks.

## Rendering and runtime

- The 20,000-node unfiltered shell remains visually dense; practical inspection
  at that scale depends on search, filters, focus, and the selected label.
- GPU timing and JavaScript heap diagnostics are only available when the runtime
  exposes those browser APIs.
- The upstream React Three Fiber path still emits a `THREE.Clock` deprecation
  warning.
