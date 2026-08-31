# Contributing

## Working rules

- Keep observation deterministic and local.
- Do not add AI/model calls or model-visible visualization reporting.
- Preserve agent stdout, stderr, exit status, and working files.
- Keep adapters failure-open and keep heavy work asynchronous and bounded.
- Do not persist source contents, credentials, databases, or dependency
  directories by default.

The implementation sequence is tracked in
[docs/progress/IMPLEMENTATION_LOG.md](docs/progress/IMPLEMENTATION_LOG.md).

## Toolchain

Validated versions are documented in [docs/TOOLCHAIN.md](docs/TOOLCHAIN.md).

Frontend dependencies live under `apps/desktop/frontend`:

```powershell
cd apps/desktop/frontend
npm ci
```

## Development workflow

Desktop development:

```powershell
cd apps/desktop
wails dev
```

Root and desktop Go validation:

```powershell
go test ./...
go vet ./...
cd apps/desktop
go test ./...
go vet ./...
```

Frontend validation:

```powershell
cd apps/desktop/frontend
npm run format:check
npm run lint
npm run typecheck
npm test -- --run
npm run build
```

Packaging:

```powershell
make package-windows
```

Pull requests run the consolidated CI workflow. Feature-branch pushes do not
run a duplicate copy. The native Windows/Linux/macOS package workflow is
reserved for `v*` release tags and explicit manual runs; run it for any release
candidate and when packaging behavior changes.

## Commit and slice hygiene

- Work one coherent `P#-S#` slice at a time.
- Prefer commit subjects that start with lowercase action verbs such as `add`,
  `upd`, or `fix`.
- Include the phase/slice ID in implementation commits.
- Run relevant checks before each commit.
- Push completed slice work before starting the next slice.

## Documentation map for contributors

- [Architecture](docs/ARCHITECTURE.md)
- [Event protocol](docs/EVENT_PROTOCOL.md)
- [Security](docs/SECURITY.md)
- [Reliability](docs/RELIABILITY.md)
- [Performance](docs/PERFORMANCE.md)
- [Codex installer](docs/CODEX_INSTALLER.md)
- [Generic wrapper](docs/GENERIC_WRAPPER.md)
- [Packaging](docs/PACKAGING.md)
- [Roadmap](docs/ROADMAP.md)

## Shared UI vendor workflow

The desktop frontend vendors `@prool-ui` packages under
`apps/desktop/frontend/vendor/prool-ui`. Do not edit installed package contents.
Use the repository-local integration guidance in
[docs/shared-ui.md](docs/shared-ui.md).
