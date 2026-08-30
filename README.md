# Agent Action Visualizer

Agent Action Visualizer (AAV) is a local-first desktop application for observing
coding-agent activity as a stable, replayable 3D project graph.

The non-negotiable project rule is that observation stays deterministic and
local. Adapters do not call a model, do not modify prompts, and do not report
visualization state back into an agent context.

## What it does

- Builds a deterministic repository graph with stable node identity.
- Tracks live focus, current/previous activity, and directional session trails.
- Renders per-access Time and Work geometry from stable access anchors.
- Persists bounded local session history for timeline replay and analytics.
- Integrates with Codex hooks and a generic wrapper without changing agent
  stdout, stderr, exit status, or working files.

## Platform status

- Windows desktop development and portable packaging are validated locally.
- The per-user NSIS installer passed install, launch, uninstall, reinstall, and
  final-cleanup validation in Windows CI and directly on the Windows validation
  host.
- Linux and macOS packaging are validated in CI only.
- Code signing has not been implemented. Windows may display unknown-publisher
  or application-reputation warnings, and macOS artifacts are not notarized.

See [known limitations](docs/KNOWN_LIMITATIONS.md) for the current release
boundary.

## Standalone Windows quickstart

Keep the desktop executable, `aav.exe`, `aav-codex-hook.exe`,
`aav-wrapper.exe`, and `SHA256SUMS.txt` together. Verify the checksums,
double-click the desktop executable, and select **Choose folder...**. The
portable app requires no developer tooling.

## Quickstart from source

Prerequisites:

- Go `1.26.6`
- Node.js `22.22.3`
- npm `10.9.8`
- Wails `2.12.0`

Install frontend dependencies:

```powershell
cd apps/desktop/frontend
npm ci
```

Run the desktop app in development mode from `apps/desktop`:

```powershell
cd ..
wails dev
```

Build the desktop app:

```powershell
wails build
```

Run the repository checks:

```powershell
go test ./...
go vet ./...
cd apps/desktop
go test ./...
go vet ./...
cd frontend
npm run format:check
npm run lint
npm run typecheck
npm test -- --run
npm run build
```

More detail, including packaging and adapter setup, is in
[docs/INSTALLATION.md](docs/INSTALLATION.md).

## Review fixture

Create a deterministic review repository with access, time, and work evidence:

```powershell
go run ./cmd/aav-review-project -out C:\tmp\aav-extrusion-review
```

Load that path in the desktop app to inspect stable shells, access points,
Time/Work geometry, filtering, analytics, and replay behavior.

## Codex and generic wrapper setup

Standalone packages already contain the three tools below. The following build
commands are only for source checkouts.

Build the CLI utilities:

```powershell
New-Item -ItemType Directory -Force .cache\aav-bin | Out-Null
go build -o .cache\aav-bin\aav.exe ./cmd/aav
go build -o .cache\aav-bin\aav-codex-hook.exe ./cmd/aav-codex-hook
go build -o .cache\aav-bin\aav-wrapper.exe ./cmd/aav-wrapper
```

Project-local Codex hook install, status, test, and uninstall:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex status --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex test --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex uninstall --scope project --project C:\path\to\repository
```

Generic wrapper usage:

```powershell
.\.cache\aav-bin\aav-wrapper.exe --project-root C:\path\to\repository -- your-agent-command argument
```

See [docs/CODEX_INSTALLER.md](docs/CODEX_INSTALLER.md) and
[docs/GENERIC_WRAPPER.md](docs/GENERIC_WRAPPER.md) for the authoritative
contracts and failure-open guarantees.

## Documentation map

- [Installation](docs/INSTALLATION.md)
- [User guide](docs/USER_GUIDE.md)
- [Timeline and replay](docs/TIMELINE_REPLAY.md)
- [Privacy and no-model-call boundary](docs/PRIVACY.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Known limitations](docs/KNOWN_LIMITATIONS.md)
- [Contributor guide](CONTRIBUTING.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Event protocol](docs/EVENT_PROTOCOL.md)
- [Security and threat model](docs/SECURITY.md)
- [Reliability and recovery](docs/RELIABILITY.md)
- [Rendering performance](docs/PERFORMANCE.md)
- [Packaging](docs/PACKAGING.md)
- [Release checklist](docs/RELEASE_CHECKLIST.md)
- [Codex compatibility evidence](docs/CODEX_COMPATIBILITY.md)
- [Codex hook behavior](docs/CODEX_HOOK.md)
- [Codex reproducible end-to-end validation](docs/CODEX_E2E.md)
- [Implementation log](docs/progress/IMPLEMENTATION_LOG.md)

## License

Licensed under Apache License 2.0. See [LICENSE](LICENSE).
