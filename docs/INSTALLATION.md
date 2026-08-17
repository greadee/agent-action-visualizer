# Installation and setup

This guide covers source builds, local desktop development, packaging, Codex
hook installation, and the generic wrapper entry point.

## Toolchain

Validated versions:

- Go `1.26.3`
- Node.js `22.22.3`
- npm `10.9.8`
- Wails `2.12.0`

See [toolchain details](TOOLCHAIN.md) for the compatibility rationale.

## Clone and install

From the repository root:

```powershell
git clone https://github.com/greadee/agent-action-visualizer.git
cd agent-action-visualizer
cd apps/desktop/frontend
npm ci
```

The frontend dependencies live under `apps/desktop/frontend`. There is no
repository-root `package.json`.

## Run the desktop app

From `apps/desktop`:

```powershell
wails dev
```

This starts the desktop shell and the Vite frontend watcher used by Wails.

## Build the desktop app

From `apps/desktop`:

```powershell
wails build
```

From the repository root, the Make target delegates to the same Wails build:

```powershell
make desktop
```

## Package release artifacts

From the repository root:

```powershell
make package-windows
```

The packaging script produces versioned Windows portable and, where the host
supports it, NSIS installer artifacts together with a SHA-256 manifest. Linux
and macOS packaging are validated in CI. See [PACKAGING.md](PACKAGING.md).

## Build CLI tools

From the repository root:

```powershell
New-Item -ItemType Directory -Force .cache\aav-bin | Out-Null
go build -o .cache\aav-bin\aav.exe ./cmd/aav
go build -o .cache\aav-bin\aav-codex-hook.exe ./cmd/aav-codex-hook
go build -o .cache\aav-bin\aav-wrapper.exe ./cmd/aav-wrapper
```

`aav.exe` is the installer/status/test CLI for the Codex integration.
`aav-codex-hook.exe` is the silent hook process Codex runs locally.
`aav-wrapper.exe` is the generic wrapper for non-Codex commands.

## Codex hook installation

Project-local install:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope project --project C:\path\to\repository
```

User-level install:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope user
```

Preview install or uninstall without writing:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope project --project C:\path\to\repository --dry-run
.\.cache\aav-bin\aav.exe codex uninstall --scope project --project C:\path\to\repository --dry-run
```

Check, test, and remove an installation:

```powershell
.\.cache\aav-bin\aav.exe codex status --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex test --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex uninstall --scope project --project C:\path\to\repository
```

The installer writes only managed `hooks.json` entries and a managed hook
binary. It does not edit prompts, model settings, or Codex policy. See
[CODEX_INSTALLER.md](CODEX_INSTALLER.md) for the exact ownership and recovery
contract, and [CODEX_E2E.md](CODEX_E2E.md) for the reproducible hook-path
validation boundary.

## Generic wrapper usage

Windows:

```powershell
.\.cache\aav-bin\aav-wrapper.exe --project-root C:\path\to\repository -- your-agent-command argument
```

macOS or Linux:

```sh
./.cache/aav-wrapper --project-root /path/to/repository -- your-agent-command argument
```

The wrapper forwards stdin, stdout, stderr, arguments, and exit behavior to the
child command. Observation failure can lose visualization evidence only. See
[GENERIC_WRAPPER.md](GENERIC_WRAPPER.md).

## Deterministic review project

From the repository root:

```powershell
go run ./cmd/aav-review-project -out C:\tmp\aav-extrusion-review
```

Open the generated repository in the desktop app to inspect stable anchors,
Time/Work geometry, replay, filtering, and analytics.

## Validation commands

Repository root:

```powershell
go test ./...
go vet ./...
```

Desktop module:

```powershell
cd apps/desktop
go test ./...
go vet ./...
```

Frontend:

```powershell
cd apps/desktop/frontend
npm run format:check
npm run lint
npm run typecheck
npm test -- --run
npm run build
```

## Verified platform boundary

- Windows desktop development and packaging are validated locally.
- Linux and macOS packaging are validated in CI, not on this host.
- The AppX-installed Codex CLI on this Windows host still returns
  `Access is denied` when invoked directly, so live authenticated Codex turns
  cannot be launched from this terminal. The documented hook protocol itself is
  validated through reproducible local fixtures and tests.
