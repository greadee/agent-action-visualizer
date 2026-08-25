# Security and threat model

This document records the P9-S3 security boundary reviewed on 2026-08-06. Agent Action Visualizer is local-first observation software. It does not require public network listeners, AI/model calls, prompt changes, or model-visible visualization reporting.

## Protected assets

- Repository files and agent-created working changes.
- Agent stdout, stderr, exit status, arguments, and environment.
- Source text, prompts, model responses, credentials, and local configuration.
- Event/session integrity and deterministic replay state.
- Availability of the observed agent and desktop application.

## Trust boundaries

Adapters and filesystem evidence are untrusted structured input. The collector, desktop process, selected project root, and current operating-system account are trusted. Other processes running as the same account are inside the current IPC trust boundary; the transport does not add an application token.

Windows IPC uses a per-user named-pipe endpoint and a protected DACL granting access to SYSTEM and the current user SID. Unix IPC creates the parent directory with mode `0700` and the socket with mode `0600`. The server accepts at most 32 concurrent connections, applies a 250 ms connection deadline, limits frames to 256 KiB and batches to 128 events, and validates the complete batch before submitting any event.

Loopback development servers are not adapter ingress and must not be exposed as production collectors. `AAV_COLLECTOR_ENDPOINT` changes the local endpoint location but does not weaken the platform ACL created by the collector.

## Threats and controls

| Threat                                                            | Control                                                                                                                             | Verification                                                  |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| Malformed, truncated, wrong-version, or partially invalid batches | Bounded framing, JSON/schema validation, metadata depth/count limits, all-or-nothing batch validation                               | `internal/ipc`, `protocol/go` adversarial tests               |
| Oversized strings or metadata causing memory/CPU pressure         | 256 KiB frame, 64 KiB metadata, 1,024 entries, depth 8, schema-aligned field bounds                                                 | Protocol and IPC tests                                        |
| Lexical path traversal                                            | Project-relative canonicalization and out-of-root rejection                                                                         | Ingest traversal tests                                        |
| Existing symlink escaping the selected project                    | Each existing path component is inspected; out-of-root symlink targets are rejected; scanners never follow symlinks                 | Ingest and scanner symlink tests on symlink-capable platforms |
| Prompt, response, environment, source, or command retention       | Raw command removal, metadata allowlist, label redaction, ingress and SQLite defense-in-depth                                       | Unique-marker persistence regression test                     |
| Terminal/control-character label injection                        | Valid UTF-8, NUL rejection, single-line label normalization, bounded UTF-8 truncation                                               | Protocol and wrapper label tests                              |
| Installer writes through a managed symlink                        | Installer uses `Lstat` and refuses symlink config, state, backup, source, and binary files                                          | Installer symlink test                                        |
| Configuration loss                                                | Pre-change digest-backed backup, atomic swap writes, interrupted-write recovery, exact restoration, unrelated-key preservation      | Installer backup/restoration tests                            |
| Collector outage or overload changing agent behavior              | Short deadlines, bounded queues, silent failure, asynchronous observation, stdout/stderr/exit preservation                          | Hook, wrapper, emitter, and disconnected-collector tests      |
| Malicious renderer text                                           | Renderer receives normalized DTOs rather than raw command/tool output; React text rendering is used                                 | Bridge contract and frontend tests                            |
| Dependency or CI drift                                            | Lockfiles/checksums, `go mod verify`, high-severity `npm audit` CI gate, read-only workflow permissions, weekly Dependabot coverage | CI and local integrity checks                                 |

## Retention audit

SQLite retains event identity, session identity, timestamp, type, project-relative path, provenance/confidence, deterministic counts/hashes, and allowlisted reducer metadata. Project records retain the selected local root because the desktop must reopen a project. Raw source, prompts, model responses, environment variables, command arguments, arbitrary adapter metadata, and credentials are not retained by default.

Codex patch bodies and tool responses are parsed only in bounded memory to derive event paths and exact deltas. The generic wrapper forwards arguments and environment to the child process but does not copy them into lifecycle events. Filesystem scanning records metadata and never reads source contents for persistence. Diff calculation may inspect permitted local evidence asynchronously; it persists counts and evidence state, not source snapshots.

## Dependency and build review

The root and desktop modules passed `go mod verify`. The initial offline npm cache reported no advisories; the network-refreshed CI audit then identified high findings in `brace-expansion` 5.0.7 and `undici` 7.28.0. The lockfile now selects patched 5.0.9 and 7.29.0 releases, and the P10-S3 registry audit reports zero findings at the enforced high threshold. The P10-S3 release scan ran `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` against both modules with Go 1.26.6 and found no reachable vulnerabilities.

CI has read-only repository permissions, verifies Go module checksums, installs npm dependencies with `npm ci`, audits high-severity npm findings, and uploads only the generated frontend bundle for one day. No workflow receives repository credentials beyond GitHub's read-only token. Action tags remain a supply-chain trust dependency and are monitored by Dependabot.

## Residual risks

- A malicious process already running as the current user can connect to the collector and spoof bounded events. Application-level IPC credentials are not implemented.
- Path checking is subject to the normal check/use race if an attacker can replace path components after validation. The visualizer never uses an event path to write repository contents.
- Symlink tests may skip on Windows hosts that do not grant symlink creation; Linux CI exercises them.
- Secret redaction is defense-in-depth, not a universal secret detector. The stronger control is dropping raw commands and non-allowlisted metadata.
- Dependencies and GitHub Actions still require upstream supply-chain trust. Release validation must refresh both npm and Go vulnerability databases.

## Reporting

Do not include source, credentials, databases, or private event payloads in a report. Provide the affected version, platform, reproduction using synthetic fixtures, and the expected security boundary through the repository's private maintainer contact or GitHub security advisory workflow.
