# Known limitations

This file records the Windows initial-release boundary as of August 30, 2026.

## Platform and packaging

- Windows production and portable packaging are validated locally from a clean
  clone. The final byte-for-byte package repeat-launch limitation is recorded in
  the [release checklist](RELEASE_CHECKLIST.md).
- NSIS is not installed on the validation host, so installer compilation remains
  a Windows CI responsibility. The current per-user installer avoids UAC and its
  downloaded CI artifact passed two complete install/launch/uninstall cycles
  locally as well as in CI.
- Code signing has not been implemented. Windows may display unknown-publisher
  or application-reputation warnings. macOS artifacts are also not notarized.
- Linux and macOS packaging are validated in CI only, not on this host.
- Desktop automation opened the native folder picker but could not address its
  owned Explorer controls. Picker result handling is covered by Wails binding
  integration tests; an actual selection inside that native dialog was not
  automated.

## Codex boundary

- The documented Codex hook protocol, installer, failure-open behavior, and
  silent observation contract are validated through reproducible local fixtures
  and tests.
- A real authenticated Codex turn cannot be launched from this Windows host
  because the installed AppX Codex CLI returns `Access is denied` when invoked
  directly.
- Project-scope Codex hooks still depend on Codex trust review and local runtime
  support outside this repository.
- Agent Action Sync remains deliberately deferred. The application makes no AI
  or model calls and does not report visualization state into agent context.

## Replay and analytics

- Replay history begins only after this build has persisted sessions for the
  selected project. Empty history is shown explicitly; geometry is not invented.
- Replay depends on the local journal remaining available.
- Analytics reflect the currently visible, filtered graph and trail only.
- Subagent activity is only as specific as the adapter-provided agent ID.

## Security and recovery boundary

- Same-user processes remain inside the current local IPC trust boundary.
- Secret redaction is defense-in-depth rather than a universal detector.
- Release validation refreshes npm and Go vulnerability data; future releases
  must repeat those network-backed checks.
- Disconnected collection deliberately fails open: the static local graph stays
  available, while activity emitted during the outage can be lost.

## Rendering and runtime

- The 20,000-node unfiltered shell remains visually dense; practical inspection
  at that scale depends on search, filters, focus, and geometry-layer toggles.
- GPU timing and JavaScript heap diagnostics are only available when the runtime
  exposes those browser APIs.
- The upstream React Three Fiber path still emits a `THREE.Clock` deprecation
  warning.
