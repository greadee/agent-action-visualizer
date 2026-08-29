# Installation and setup

This guide covers the standalone Windows release, source builds, Codex hook
installation, and the generic wrapper entry point.

## Standalone Windows setup

Download or extract the complete portable release set into one directory:

- `agent-action-visualizer-v0.1.0-windows-amd64.exe`
- `aav.exe`
- `aav-codex-hook.exe`
- `aav-wrapper.exe`
- `SHA256SUMS.txt`

Verify the checksums, then double-click the desktop executable. No Go, Node.js,
npm, Wails, terminal, or source checkout is required. On first run, select
**Choose folder...** and choose a local Git repository. The app builds the
deterministic local graph even when no collector is connected.

The current executables are unsigned, so Windows may show an unknown-publisher
warning. Compare the release files with `SHA256SUMS.txt` before continuing.

The NSIS installer installs the same desktop executable and three adjacent setup
tools for the current user under
`%LocalAppData%\Programs\greadee\Agent Action Visualizer`. It does not
request administrator elevation. It creates current-user Start menu and desktop
shortcuts and registers the uninstaller in Windows **Installed apps**.

Uninstall removes the application files, shortcuts, and registration while
preserving the separate local session journal and preferences for recovery or
reinstallation. See the [release checklist](RELEASE_CHECKLIST.md) for the
validated install, launch, uninstall, and reinstall lifecycle.

## Toolchain

Validated versions:

- Go `1.26.6`
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

## Build CLI tools from source

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

For the portable or installed release, run these commands from the directory
containing the application. Project-local install:

```powershell
.\aav.exe codex install --scope project --project C:\path\to\repository
```

User-level install:

```powershell
.\aav.exe codex install --scope user
```

Preview install or uninstall without writing:

```powershell
.\aav.exe codex install --scope project --project C:\path\to\repository --dry-run
.\aav.exe codex uninstall --scope project --project C:\path\to\repository --dry-run
```

Check, test, and remove an installation:

```powershell
.\aav.exe codex status --scope project --project C:\path\to\repository
.\aav.exe codex test --scope project --project C:\path\to\repository
.\aav.exe codex uninstall --scope project --project C:\path\to\repository
```

The installer writes only managed `hooks.json` entries and a managed hook
binary. It does not edit prompts, model settings, or Codex policy. See
[CODEX_INSTALLER.md](CODEX_INSTALLER.md) for the exact ownership and recovery
contract, and [CODEX_E2E.md](CODEX_E2E.md) for the reproducible hook-path
validation boundary.

## Generic wrapper usage

Windows:

```powershell
.\aav-wrapper.exe --project-root C:\path\to\repository -- your-agent-command argument
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

- Windows desktop development and portable packaging are validated locally.
- The per-user NSIS installer is built in Windows CI and its downloaded artifact
  passed install, launch, uninstall, reinstall, and final-cleanup validation on
  this host.
- Release executables are unsigned.
- Linux and macOS packaging are validated in CI, not on this host.
- The AppX-installed Codex CLI on this Windows host still returns
  `Access is denied` when invoked directly, so live authenticated Codex turns
  cannot be launched from this terminal. The documented hook protocol itself is
  validated through reproducible local fixtures and tests.
