# Implementation log

## P0-S1 — Repository / bootstrap

- Phase: 0
- Slice: P0-S1
- Feature: Repository
- Action: Bootstrap
- Status: complete
- Files changed: `README.md`, `LICENSE`, `.gitignore`, `.editorconfig`, `AGENTS.md`, `docs/progress/IMPLEMENTATION_LOG.md`
- Tests run: repository-state and toolchain inspection only
- Benchmark result: not applicable
- Commit SHA: `c223edf`
- Known limitations: application code is intentionally deferred to P0-S3; Wails remains to be installed
- Next slice: P0-S2 — Architecture / decisions

## P0-S2 — Architecture / decisions

- Phase: 0
- Slice: P0-S2
- Feature: Architecture
- Action: Define
- Status: complete
- Files changed: `docs/ARCHITECTURE.md`, `docs/ROADMAP.md`, `docs/adr/*`, `docs/progress/IMPLEMENTATION_LOG.md`
- Tests run: `git diff --check`; verified 14 numbered ADRs contain the required decision sections
- Benchmark result: not applicable
- Commit SHA: `9b3cf88`
- Known limitations: decisions will be validated by implementation spikes and superseded through new ADRs when evidence changes
- Next slice: P0-S3 — Toolchain / scaffold

## P0-S3 — Toolchain / scaffold

- Phase: 0
- Slice: P0-S3
- Feature: Toolchain
- Action: Scaffold
- Status: complete
- Files changed: Go workspace, Wails desktop shell, React/TypeScript frontend, lockfiles, lint/test/build configuration, CI, toolchain documentation
- Tests run: Go unit tests and vet for root/desktop; Prettier check; ESLint; TypeScript check; Vitest (1 test); Vite production build; Wails Windows production build
- Benchmark result: not applicable
- Commit SHA: `31ad215`
- Known limitations: platform packaging is deferred to P10; the initial Three.js bundle is 1.07 MB before later code splitting/LOD work; the first scaffold screen is not the production renderer
- Next slice: P1-S1 — Protocol / define

## P1-S1 — Protocol / define

- Phase: 1
- Slice: P1-S1
- Feature: Protocol
- Action: Define
- Status: complete
- Files changed: versioned JSON Schema, Go and TypeScript types, fixtures, compatibility tests, protocol documentation
- Tests run: `go test ./...` including round-trip, invalid-event, unknown-field, and cross-language enum contract tests; frontend TypeScript, ESLint, and Prettier checks
- Benchmark result: not applicable
- Commit SHA: `15bdbdf`
- Known limitations: generated-code automation is deferred; types are hand-maintained and contract-tested
- Next slice: P1-S2 — Ingestion / implement

## P1-S2 — Ingestion / implement

- Phase: 1
- Slice: P1-S2
- Feature: Ingestion
- Action: Implement
- Status: complete
- Files changed: bounded priority queue, asynchronous collector, path normalization, deduper, tests/benchmark, mock-event CLI
- Tests run: `go test ./...`; mock-event CLI validated against the normalized v1 fixture
- Benchmark result: `BenchmarkQueueOffer` 21.36 ns/op on Windows/amd64 Ryzen 5 9600X (100 ms benchtime); queue-only measurement, not persistent IPC
- Commit SHA: `4720e54`
- Known limitations: native pipe/socket transport is deferred to its platform slice; this slice establishes the transport-independent collector boundary
- Next slice: P1-S3 — Session / implement

## P1-S3 — Session / implement

- Phase: 1
- Slice: P1-S3
- Feature: Session
- Action: Implement
- Status: complete
- Files changed: session lifecycle, focus priority, previous/secondary state, access intervals, idle handling, deterministic tests
- Tests run: `go test ./...` including active/previous transition, same-timestamp write priority, stable rename identity, session stop, and idle-timeout tests
- Benchmark result: not applicable
- Commit SHA: `db2a774`
- Known limitations: persistence/recovery follows in P1-S4; simultaneous multi-agent focus policy will be expanded with adapter identities
- Next slice: P1-S4 — Persistence / implement

## P1-S4 — Persistence / implement

- Phase: 1
- Slice: P1-S4
- Feature: Persistence
- Action: Implement
- Status: complete
- Files changed: pinned pure-Go SQLite dependency, initial migration, event/session repositories, recovery and restart tests
- Tests run: `go test ./...` including migrations, idempotent event persistence, database reopen, and active-session recovery
- Benchmark result: not applicable
- Commit SHA: `b8741a9`
- Known limitations: graph/node repositories are schema-ready and will be filled with P2 graph behavior
- Next slice: P2-S1 — Scanner / implement

## P2-S1 — Scanner / implement

- Phase: 2
- Slice: P2-S1
- Feature: Scanner
- Action: Implement
- Status: complete
- Files changed: root-constrained scanner, default/.gitignore/.aavignore filters, classification, symlink handling, tests
- Tests run: `go test ./...` including default, `.gitignore`, `.aavignore`, and file classification behavior
- Benchmark result: not applicable
- Commit SHA: `2b89892`
- Known limitations: gitignore negation and the full gitignore grammar are deferred to a dedicated matcher replacement
- Next slice: P2-S2 — Identity / stabilize

## P2-S2 — Identity / stabilize

- Phase: 2
- Slice: P2-S2
- Feature: Identity
- Action: Stabilize
- Status: complete
- Files changed: deterministic node IDs, case/separator policy, rename aliases, tombstone lifecycle, tests
- Tests run: `go test ./...` including deterministic IDs, case/separator normalization, rename alias, and tombstone tests; CI exposed and a follow-up fix canonicalized Windows separators before OS-specific cleaning
- Benchmark result: not applicable
- Commit SHA: `6f61925`
- Known limitations: cross-rescan content-based rename correlation is deferred; exact adapter renames preserve identity now
- Next slice: P2-S3 — Layout / implement

## P2-S3 — Layout / implement

- Phase: 2
- Slice: P2-S3
- Feature: Layout
- Action: Implement
- Status: complete
- Files changed: deterministic spherical hierarchy, stable prior-position reuse, structure edges, snapshots/patches, tests
- Tests run: `go test ./...` including deterministic layout, position preservation after additions, hierarchy edges, rename patch behavior
- Benchmark result: not applicable
- Commit SHA: `40e47b8`
- Known limitations: collision refinement and large-graph LOD are deferred to P9; default positions prioritize stability
- Next slice: P3-S1 — Renderer / establish

## P3-S1 — Renderer / establish

- Phase: 3
- Slice: P3-S1
- Feature: Renderer
- Action: Establish
- Status: complete
- Files changed: graph types/sample, instanced nodes, buffered hierarchy edges, orbit controls, hover/selection labels, UI test
- Tests run: Prettier, ESLint, TypeScript, Vitest, and Vite production build
- Benchmark result: not applicable
- Commit SHA: `9bc79fb`
- Known limitations: the scene uses deterministic fixture data until the Go bridge slice; camera focus follows in P3-S2

## P3-S2 - Camera / focus

- Phase: 3
- Slice: P3-S2
- Feature: Camera
- Action: Focus
- Status: complete
- Files changed: bounded focus math, quaternion camera controller, manual interruption, recenter action, reduced-motion behavior, formula documentation, unit tests
- Tests run: Prettier, ESLint, TypeScript, Vitest (4 tests), and Vite production build
- Benchmark result: not applicable
- Commit SHA: `4a99d20`
- Known limitations: live-event Auto-follow is intentionally deferred to P4-S2; camera behavior is unit-tested mathematically and awaits browser interaction automation in P9
- Next slice: P3-S3 - UI / inspect

## P3-S3 - UI / inspect

- Phase: 3
- Slice: P3-S3
- Feature: UI
- Action: Inspect
- Status: complete
- Files changed: project/search controls, layout and visibility controls, shared node palette, graph legend, selected-node inspector, responsive panel styling, UI assertions
- Tests run: Prettier, ESLint, TypeScript, Vitest (4 tests), and Vite production build
- Benchmark result: production bundle 1,104.67 kB (304.22 kB gzip); code splitting remains a P9 optimization
- Commit SHA: `f65c69b`
- Known limitations: activity values correctly report not observed until P4 live events supply evidence; timeline, activity spikes, and session trail belong to later phases
- Next slice: P4-S1 - Bridge / stream

## P4-S1 - Bridge / stream

- Phase: 4
- Slice: P4-S1
- Feature: Bridge
- Action: Stream
- Status: complete
- Files changed: Wails project load/refresh API, metadata-only DTOs, graph snapshot/patch events, revision-aware frontend reducer/subscription, project-path control, cross-layer tests, bridge documentation
- Tests run: core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (5 tests), and Vite production build
- Benchmark result: not applicable
- Commit SHA: `de38b6f`
- Known limitations: refresh is an explicit API call until filesystem/adaptor triggers arrive; native directory-picker UX is deferred while the path-based API remains functional
- Next slice: P4-S2 - Focus / animate

## P4-S1A - Graph readability / stabilize

- Phase: 4 corrective stabilization
- Slice: P4-S1A
- Feature: Git involvement shell and activity edit points
- Action: Stabilize
- Status: complete
- Files changed: Git history enrichment, agent activity report loader, uniform-radius grouped layout, Time/Work extrusion renderer, populated inspector, review-project generator, geometry documentation, cross-layer tests
- Tests run: complete core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (8 tests), Vite production build; actual Wails bridge browser review in Time and Work modes
- Benchmark result: production bundle 1,109.80 kB (305.74 kB gzip); rendering remains instanced for nodes and edit points
- Commit SHA: `1839161`
- Known limitations: Git rename inference is deliberately not claimed; agent report ingestion is file-based until native adapters publish equivalent events
- Next slice: paused by user until this graph model is accepted

## P4-S2 - Focus / animate

- Phase: 4
- Slice: P4-S2
- Feature: Focus
- Action: Animate
- Status: complete
- Files changed: session focus metadata, root-constrained Wails activity publisher, current/previous/secondary focus styling, Auto-follow/manual inactivity/pause/return-live controls, deterministic focus-review fixture, bridge documentation, and cross-layer tests
- Tests run: complete core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (10 tests), Vite production build, Wails Windows production build; in-app browser review of focus transitions, marker overlap priority, pause/return-live, and inactivity resume
- Benchmark result: production bundle 1,113.29 kB (306.86 kB gzip); rendering remains instanced and focus changes update existing instances
- Commit SHA: `823e8d2`
- Known limitations: native adapters must still call `PublishActivityEvent`; the ordered session trail belongs to P4-S3 and was intentionally not started
- Next slice: paused per user; do not proceed beyond P4-S2

## P4-S3 - Trail / visualize

- Phase: 4
- Slice: P4-S3
- Feature: Trail
- Action: Visualize
- Status: complete
- Files changed: metadata-only access-trail DTOs, same-path interval reopening after idle, deterministic recency/complete-session selection, buffered recency-colored travel edges, instanced direction markers, hover metadata, older-access node styling, independent controls/legend, documentation, and cross-layer tests
- Tests run: complete core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (15 tests), Vite production build, Wails Windows production build; in-app browser review of six ordered accesses, last-three limit, complete-session mode, independent structure/trail visibility, and marker hover metadata
- Benchmark result: production bundle 1,118.85 kB (308.48 kB gzip); travel lines use one buffer and direction markers remain instanced
- Commit SHA: `b4d163e`
- Known limitations: transitions whose nodes are absent from the current graph are omitted; the replay cursor/timeline belongs to P8, and per-access activity points begin in P5-S1
- Next slice: P5-S1 - Access points / render

## P5-S1 - Access points / render

- Phase: 5
- Slice: P5-S1
- Feature: Access points
- Action: Render
- Status: complete
- Files changed: deterministic per-access surface anchors, dense-history aggregation, instanced access-point renderer and hover metadata, independent control and legend, dense review fixture, documentation, and unit/UI tests
- Tests run: complete core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (18 tests), Vite production build, Wails Windows production build; in-app browser review of a 60-access dense session and the independent access-points control
- Benchmark result: production bundle 1,122.04 kB (309.36 kB gzip); points remain instanced and dense histories compact older accesses into aggregate markers
- Commit SHA: `a9cdae1`
- Known limitations: points carry access metadata only; time-duration growth/scaling is P5-S2 and work/diff representation is P5-S3
- Next slice: P5-S2 - Time mode / render

## P5-S2 - Time mode / render

- Phase: 5
- Slice: P5-S2
- Feature: Time mode
- Action: Render
- Status: complete
- Files changed: deterministic per-access time-extrusion geometry, bounded linear/logarithmic scale helpers, live active-interval growth capped at the session idle limit, exact-duration tooltip and selected-access inspector treatment, duration review fixture, documentation, and unit/UI tests
- Tests run: core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (24 tests), Vite production build, Wails Windows production build; in-app browser review of short, medium, clamped, and active duration intervals in Time mode
- Benchmark result: production bundle 1,126.39 kB (310.42 kB gzip); time geometry uses one buffered line set and an instanced endpoint mesh
- Commit SHA: `f898de3`
- Known limitations: scale selection and user-facing clamp controls are deferred to P5-S4; work additions/deletions remain deferred to P5-S3
- Next slice: P5-S3 - Work mode / calculate

## P5-S3 - Work mode / calculate

- Phase: 5
- Slice: P5-S3
- Feature: Work mode
- Action: Calculate
- Status: complete
- Files changed: bounded asynchronous line-delta pipeline with structured-patch, correlated-snapshot, and batched Git evidence; per-access work provenance; outward addition and inward deletion geometry; explicit empty, unknown, binary, unsupported-encoding, and pending markers; exact tooltip/inspector values; review fixtures; documentation; and cross-layer tests
- Tests run: complete core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (31 tests), Vite production build, Wails Windows production build; in-app browser review of addition, deletion, mixed, empty, unknown, binary, and extreme deltas plus exact hover and inspector evidence
- Benchmark result: production bundle 1,131.94 kB (312.00 kB gzip); the desktop pipeline uses a 64-request queue, 16-request Git batches, a 2-second Git timeout, bounded completed-key deduplication, and non-blocking fail-open submission
- Commit SHA: `299a9c6`
- Known limitations: user-facing work scale and clamp controls are deferred to P5-S4; Git fallback reflects working-tree state when its batch executes and reports unknown when no exact row is available; native adapters do not provide correlated snapshots until their later phases
- Next slice: P5-S4 - Activity controls / refine

## P5-S4 - Activity controls / refine

- Phase: 5
- Slice: P5-S4
- Feature: Activity controls
- Action: Refine
- Status: complete
- Files changed: shared activity display settings, accessible Time/Work scale and mode-specific cap controls, cap-aware time/work renderers, exact-value clamp treatment in tooltips and inspector, legend state, deterministic cross-mode fixtures, documentation, and regression tests
- Tests run: complete core and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, Wails Windows production build, and `npm ci`; browser review of time, work, linear/logarithmic scaling, duration/work caps, mixed and extreme work, and dense history fixtures
- Benchmark result: production bundle 1,134.24 kB (312.75 kB gzip); the existing code-splitting warning remains deferred to P9
- Commit SHA: `4f589a9`
- Known limitations: display preferences are session-local until P8 persisted preferences; the Three.js `Clock` deprecation warning is upstream and does not affect geometry behavior
- Phase summary: every access retains a stable point; time and additions extend outward, deletions extend inward, and exact duration and work evidence remains inspectable after visual clamping across deterministic Time and Work replay
- Next slice: P6-S1 - Codex research / verify

## P6-S1 - Codex research / verify

- Phase: 6
- Slice: P6-S1
- Feature: Codex integration
- Action: Research and verify
- Status: complete
- Files changed: version-specific Codex compatibility evidence and sanitized lifecycle-hook/App Server contract fixtures; no adapter, hook configuration, or installation mutation
- Tests run: fixture JSON parse; core and desktop Go tests/vet; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, and Wails Windows production build
- Benchmark result: not applicable; this verification-only slice introduces no runtime adapter. Production bundle remains 1,134.24 kB (312.75 kB gzip)
- Commit SHA: `8df031c`
- Known limitations: the installed Windows package directory identifies version `26.721.4979.0`, but this managed environment denied direct CLI execution; no trusted project hook was configured to exercise live payloads. Hooks do not cover every tool path, and App Server is not a documented passive existing-session observer
- Next slice: P6-S2 - Codex hook / implement

## P6-S2 - Codex hook / implement

- Phase: 6
- Slice: P6-S2
- Feature: Codex lifecycle adapter
- Action: Implement
- Status: complete
- Files changed: silent bounded Codex command-hook executable; metadata-only lifecycle/tool translation; structured `apply_patch` and recognized path extraction; deterministic correlation/deduplication; per-user Windows named-pipe and Unix-socket transport; bounded desktop ingress integration; sanitized fixtures; adapter, transport, subprocess, and desktop integration tests; compatibility and benchmark documentation
- Tests run: complete root and desktop Go tests/vet; repeated hook/IPC tests; malformed, oversized, blocked-input, blocked-sender, disconnected-collector, silent-output, exit-code, path, delta, and process-level contract tests; Linux IPC test cross-compilation and hook cross-build; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, and Wails Windows production build
- Benchmark result: Windows/amd64 on AMD Ryzen 5 9600X with Go 1.26.3: five-run median 12.575 microseconds for bounded decode/translation/no-op send, 73.212 microseconds for a named-pipe round trip, and 6.890 milliseconds for the process-level hook round trip across 20 launches per run
- Commit SHA: `baa877d`
- Known limitations: installer/status/uninstall and Codex trust/configuration mutation are deferred to P6-S3; live installed-Codex payload and end-to-end overhead validation remain P6-S4; hosted and ambiguous tool paths intentionally produce no file semantics; race tests were unavailable because the Windows Go environment has CGO disabled; `npm ci` continues to report one existing high-severity audit advisory
- Next slice: P6-S3 - Codex installer / implement

## P6-S3 - Codex installer / implement

- Phase: 6
- Slice: P6-S3
- Feature: Codex installer
- Action: Implement
- Status: complete
- Files changed: project-local and user-level `hooks.json` installer; install, uninstall, status, isolated test, and dry-run CLI surfaces; marked six-event hook merging; exact first-install backups; state and binary SHA-256 validation; recoverable same-directory writes; idempotent reinstall; Windows/POSIX quoting; bounded diagnostics; runtime ignore rules; isolated temporary-fixture tests; and installer documentation
- Tests run: complete root and desktop Go tests/vet; installer tests for exact restoration, unrelated post-install changes, project/user scope, dry-run, idempotency, invalid JSON, backup/state mismatch, interrupted writes, unmanaged files, binary tampering, hook layout, and path quoting; executable-level project and user install/test/uninstall from Windows paths with spaces; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, and Wails Windows production build
- Benchmark result: not applicable; the installer is operator-invoked rather than an agent critical-path component. The installed-hook diagnostic completed successfully with exit code zero, empty stdout/stderr, and one sanitized event delivered over an isolated Windows named pipe
- Commit SHA: `ebad63ecdb524cfd05b3175fcb1104a74d4b5f5d`
- Known limitations: Codex project trust and hook review were not mutated or inferred; a real trusted Codex session remains P6-S4. The installer deliberately leaves unrelated inline `config.toml` hooks untouched, so Codex may warn when both representations exist in one layer. Executable-level installer validation was performed on Windows/amd64; macOS and Linux paths are covered by platform-neutral unit contracts but were not run on those hosts
- Next slice: P6-S4 - Codex E2E / validate

## P6-S4 - Codex E2E / validate

- Phase: 6
- Slice: P6-S4
- Feature: Codex end-to-end validation
- Action: Validate
- Status: complete
- Files changed: reproducible documented-hook Git-fixture session test; hook-process, local IPC, normalized event, focus, structured-work, disconnect/reconnect, SQLite reopen, and deterministic replay validation; stable move identity and inspectable delete-tombstone continuity across desktop refresh; current compatibility and E2E evidence documentation
- Tests run: complete root and desktop Go tests/vet; explicit reproducible fixture and move/tombstone commands; hook/IPC/process benchmark suites; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, and Wails Windows production build
- Benchmark result: Windows/amd64 on AMD Ryzen 5 9600X with Go 1.26.3: five-run medians were 11.866 microseconds for bounded decode/translation/no-op send, 51.933 microseconds for a named-pipe round trip, and 7.553 milliseconds for the process-level hook invocation across 20 launches per run
- Commit SHA: `4f233d23c81b75245395164f1503064051162e19`
- Known limitations: the installed AppX Codex `26.721.4979.0` CLI returns `Access is denied` for both restricted and elevated `--version`/`--help` probes. This host therefore cannot run `codex exec`, review a project hook with `/hooks`, or produce a real authenticated session. P6-S4 validates the documented local hook contract without model execution; a real trusted-session check needs an executable Codex host. macOS and Linux remain unverified.
- Phase summary: Phase 6 now has version-specific compatibility evidence, a silent bounded lifecycle adapter, reversible project/user installation, and a reproducible full local hook session that validates lifecycle, read/create/patch/move/delete, focus, closed time intervals, structured additions/deletions, disconnected collector recovery, persistence/replay readiness, and no model-visible hook output. A live CLI session is honestly documented as unverified on this Windows AppX environment.
- Next slice: P7-S1 - Adapter SDK / define

## P7-S1 - Adapter SDK / define

- Phase: 7
- Slice: P7-S1
- Feature: Public adapter SDK
- Action: Define
- Status: complete
- Files changed: public Go adapter descriptor, capability declarations, static contract validation, bounded non-blocking failure-open emitter, mock collector, safe project-root path normalization, runnable metadata-only example adapter, contract tests, and adapter-author guidance; ingress now shares the public path normalizer
- Tests run: root and desktop Go tests/vet; SDK race attempt was unavailable because this Windows Go environment has CGO disabled; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), and Vite production build
- Benchmark result: not applicable; the SDK bounds batches at 128 events, sends at four in-flight collector calls by default, and uses a 75-millisecond local collector deadline
- Commit SHA: `b6c16109ca74e73d7d0356359094f59365e98541`
- CI follow-up SHA: `eda39b5`; waits for the bounded local IPC fixture delivery before asserting the Codex session lifecycle, validated by 20 consecutive fixture runs
- Known limitations: the generic process wrapper, structured-stream runner, filesystem fallback, and optional adapters are deferred to later Phase 7 slices; no visible UI changed; the existing Vite bundle-size warning remains deferred to P9
- Next slice: P7-S2 - Generic wrapper / implement

## P7-S2 - Generic wrapper / implement

- Phase: 7
- Slice: P7-S2
- Feature: Generic command wrapper
- Action: Implement
- Status: complete
- Files changed: public transparent process runner; standalone `aav-wrapper` CLI; exact argument, environment, working-directory, stdin, stdout, stderr, exit-code, and Unix signal contracts; ordered metadata-only session/command lifecycle events; bounded local delivery; bounded structured-stream parser interface with drop signaling; asynchronous filesystem/Git fallback observer seam; operational documentation; and process-level contract tests
- Tests run: complete root and desktop Go tests/vet; ten repeated wrapper and CLI contract runs; executable smoke with exit code 17; Windows process tests for success, exit code 23, quoting, environment, working directory, cancellation, disconnected collector, blocked parser/fallback, and 2 MiB stdout plus 2 MiB stderr; Linux/amd64 test cross-compilation including Unix signal forwarding/re-raising; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), and Vite production build
- Benchmark result: not applicable; wrapper observation uses a 64-batch delivery queue, 75-millisecond local send deadline, 64-chunk parser queue, 32 KiB stream chunks, and a 25-millisecond parser-drain cap
- Commit SHA: `28a8587170ad09bba52204b49d9aa1e488d38cef`
- Known limitations: the standalone CLI emits lifecycle evidence only; agent-specific structured parsers remain explicit embedding integrations, filesystem/Git observation is deferred to P7-S3, and macOS signal behavior is cross-platform code-covered but not run on this Windows host; the existing Vite bundle-size warning remains deferred to P9
- Next slice: P7-S3 - Filesystem fallback / implement

## P7-S3 - Filesystem fallback / implement

- Phase: 7
- Slice: P7-S3
- Feature: Filesystem fallback
- Action: Implement
- Status: complete
- Files changed: shared scanner/live ignore matcher with runtime reload; recursive cross-platform filesystem observer; bounded raw and pending queues; debounce and per-path coalescing; Git status and numstat batching; unique-evidence rename/move correlation; explicit observed/correlated confidence; binary and unknown work handling; cross-source structured/native duplicate suppression; default generic-wrapper CLI integration with opt-out; operational documentation; and deterministic burst, overload, long-session, real-watcher, real-Git, and child-process integration tests
- Tests run: complete root and desktop Go tests/vet; five repeated Git/watcher/dedup contract runs; ten repeated watcher and wrapper concurrency/integration runs; real Windows recursive filesystem notification; disposable real-Git status/numstat inspection; child-owned file write preservation; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, and Wails Windows production build
- Benchmark result: not applicable; the fallback uses a 1,024-event raw queue, 512 pending paths, 128-path batches, 75-millisecond debounce, 150-millisecond rename window, 250-millisecond Git deadline, and 100-millisecond cross-source evidence window
- Commit SHA: `6ea71212bc2cfe1dbcad47d7c18531687b1ccbbc`
- CI follow-up SHA: `8c5bdd286e3859c7150c97c3e243eea609800461`; defines observer readiness after recursive watch installation and raw-event drain startup so the real-watcher contract is scheduler-independent on Linux and Windows
- Known limitations: OS watcher delivery can lose evidence during kernel overflow or on unsupported/network filesystems; the shared ignore matcher preserves the established subset and does not implement negation rules; directory rename correlation requires Git evidence; session-window attribution cannot identify a specific child tool without native or structured evidence; race builds were unavailable because this Windows host has CGO disabled and no C compiler; macOS and Linux watcher behavior is implementation-covered but not run on those hosts; the existing Vite bundle-size warning remains deferred to P9
- Next slice: P7-S4 - Claude adapter / implement where stable

## P7-S4 - Claude adapter / implement where stable

- Phase: 7
- Slice: P7-S4
- Feature: Optional Claude Code adapter
- Action: Verify compatibility boundary
- Status: complete
- Files changed: current primary-source Claude Code hook/settings compatibility record and sanitized schema-shaped session and post-tool fixtures; no adapter executable, installer, hook configuration, project trust, or Claude settings mutation
- Tests run: fixture JSON parse and placeholder audit; complete root and desktop Go tests/vet; `npm ci`; Prettier, ESLint, TypeScript, Vitest (34 tests), Vite production build, and Wails Windows production build
- Benchmark result: not applicable; no Claude runtime path was installed or invoked
- Commit SHA: `cf5b3a84c3dbf0c4943e95bb5ba0a7959dd82f32`
- Known limitations: official command hooks are documented, but this Windows host has no `claude` executable, Claude settings directory, global CLI package, Claude/Anthropic environment key, or WSL installation. No version, hook payload, installer behavior, stdout/stderr behavior, exit-code preservation, or real session was validated. Claude support remains optional and unverified; future work requires an installed executable and disposable trusted project validation. The existing Vite bundle-size warning remains deferred to P9.
- Phase summary: Phase 7 now provides a stable public adapter contract, a transparent generic process wrapper, bounded recursive filesystem/Git fallback observation, and honest Claude Code compatibility evidence. All implemented adapters preserve local-only, failure-open observation; the Claude surface is documented but deliberately not enabled without a verified local runtime.
- Next slice: P8-S1 - Timeline / implement

## P8-S1 - Timeline / implement

- Phase: 8
- Slice: P8-S1
- Feature: Persisted session timeline and deterministic replay
- Action: Implement
- Status: complete
- Files changed: bounded asynchronous local SQLite session journal; current-project session selection; pure cursor-bounded replay reconstruction including persisted work results; replay timeline, scrubber, playback speed, next/previous access controls, and return-to-live behavior; preview fixtures, desktop integration tests, and frontend accessibility tests
- Tests run: `npm ci`; complete root and desktop Go tests/vet; Prettier, ESLint, TypeScript, Vitest (37 tests), Vite production build, Wails Windows production build, and direct browser review of empty history, replay selection, time geometry, access navigation boundaries, and return to live
- Benchmark result: not applicable; persistence runs through a bounded 256-record asynchronous queue with one-second local database deadlines and drops replay history rather than blocking observation when unavailable or saturated
- Commit SHA: `4ef9dcf`
- Known limitations: replay history begins only after this build is used, and depends on the local journal remaining available; existing sessions from before P8-S1 are not reconstructed. The existing Vite bundle-size warning and three pre-existing npm audit advisories remain deferred to P9.
- Next slice: P8-S2 - Filtering / implement

## P8-S2 - Filtering / implement

- Phase: 8
- Slice: P8-S2
- Feature: Deterministic visualization filtering and preferences
- Action: Implement
- Status: complete
- Files changed: pure frontend graph and access filter engine; path/directory, file type, operation, confidence, agent, and time-range controls; filtered search and focus integration; local visualization preference persistence; reset and empty-state behavior; combined-filter and rendered UI coverage
- Tests run: `npm ci`; complete root and desktop Go tests/vet with slice-local Go caches after the host cache returned access denied; Prettier, ESLint, TypeScript, Vitest (42 tests), Vite production build, Wails Windows production build, and `git diff --check`
- Visual or E2E validation: direct browser validation of accessible filter controls, generated review accesses, combined path/operation/confidence/agent filtering, filtered-focus feedback, search constrained to the visible graph, reset behavior, and visual layout
- Benchmark result: not applicable; filtering is pure memoized frontend state and does not affect observation, persistence, adapters, or model context
- Commit SHA: `7bf349338e3c8cecda3fc55f1b77ea802bbdfb40`
- Known limitations: preferences are local UI storage and are not shared between machines; relative time windows use the replay cursor during replay and wall-clock time while live; the existing Vite bundle-size warning, Three.js `Clock` deprecation warning, and pre-existing npm audit advisories remain deferred to P9
- Next slice: P8-S3 - Session analytics / implement
