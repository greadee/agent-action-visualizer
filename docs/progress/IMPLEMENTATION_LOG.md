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
