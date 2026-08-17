# Release checklist

This checklist records the P10-S3 release-candidate validation performed on
2026-08-17. It applies to version `0.1.0` at code revision
`fb04f32c6a4d628a29819625f64faf4660febe22` before the release-record commits.

## Local release gates

- [x] Root module: `go mod verify`, `go test ./...`, and `go vet ./...` with
      Go `1.26.6`.
- [x] Desktop module: `go mod verify`, `go test ./...`, and `go vet ./...` with
      Go `1.26.6` after its embedded frontend build.
- [x] Protocol compatibility and adapter contracts through the complete root Go
      suite.
- [x] Frontend: fresh `npm ci`, Prettier, ESLint, TypeScript, 59 Vitest tests,
      and the Vite production build using Node `22.22.3` and npm `10.9.8`.
- [x] Wails `2.12.0` Windows production build.
- [x] Portable Windows package build from a clean clone using
      `scripts/package.ps1 -SkipInstaller`.
- [x] Package output:
      `agent-action-visualizer-v0.1.0-windows-amd64.exe` (16,829,440 bytes),
      SHA-256 `fb00223bf7a18d0da62657c3087d0de84bc5977e1c9f62791a2c6b63a6ea5a6f`.
- [x] Clean-clone validation under the host's normal `core.autocrlf=true`
      configuration. `.gitattributes` now enforces LF checkout for repository text,
      so the fresh clone passes the frontend formatting contract.
- [x] Codex installer lifecycle and generic-wrapper process preservation remain
      covered by the P10-S2 reproducible command validation and their contract
      suites.
- [x] Security: refreshed `npm audit` reports zero findings at the enforced
      high threshold; `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
      reports no reachable vulnerabilities for either Go module.
- [x] Tracked-content audit: no credentials, local machine paths, databases,
      release binaries, dependency directories, or source snapshots are tracked.
      Secret-like text is limited to synthetic redaction/retention tests.
- [x] Generated-output audit: packaging output, databases, caches, frontend
      dependencies, and desktop binaries remain ignored. The tracked
      `apps/desktop/build` files are Wails source resources.

## Remote release gates

- [ ] Push the final release-validation and implementation-log commits.
- [ ] Confirm the push and pull-request GitHub Actions workflows pass for the
      final head.
- [ ] Open a draft pull request from `codex/initial-build` into `main`.
- [ ] Keep the draft unmerged and do not publish a release artifact.

## Release boundary

The MVP is complete: it provides deterministic local visualization, stable
access anchors with Time and Work geometry, replay, filtering, analytics,
local-only adapters, and Windows packaging. It is production-ready for the
validated Windows boundary once the remote checks pass. Linux and macOS native
packages are CI-only on this release candidate, artifacts are unsigned, macOS
is not notarized, and direct authenticated Codex sessions remain unverified on
this managed host. See [Known limitations](KNOWN_LIMITATIONS.md) for the full
boundary.
