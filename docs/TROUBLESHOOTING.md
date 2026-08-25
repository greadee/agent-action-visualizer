# Troubleshooting

## `go run` or `go test` fails with `Access is denied` under the default Go cache

This host has previously returned `Access is denied` for builds under the user
Go cache. Use a repository-local cache for the command:

```powershell
$env:GOCACHE = (Resolve-Path .cache\go-build).Path
$env:GOMODCACHE = (Resolve-Path .cache\go-mod).Path
go test ./...
```

If the cache directories do not exist yet, create them first:

```powershell
New-Item -ItemType Directory -Force .cache\go-build, .cache\go-mod | Out-Null
```

## `npm ci` fails from the repository root

The frontend package lives under `apps/desktop/frontend`. Run:

```powershell
cd apps/desktop/frontend
npm ci
```

There is no repository-root `package.json`.

## `wails dev` or `wails build` is not found

Install Wails `2.12.0` and ensure `wails` is on `PATH`. The validated toolchain
is listed in [TOOLCHAIN.md](TOOLCHAIN.md).

## The desktop app loads but no live activity appears

Check the following:

- the project path points to the intended repository root
- the collector is running as part of the desktop app
- for Codex integration, the hook is installed for the correct scope
- for project-scope Codex hooks, the project has been trusted by Codex

Use the installer diagnostics to verify the hook state:

```powershell
.\.cache\aav-bin\aav.exe codex status --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex test --scope project --project C:\path\to\repository
```

`status` can report `not-installed`, `installed`, `partial`, or `invalid`.

## Codex hook install succeeds but live Codex validation is still incomplete

That is expected on this Windows host. The installed AppX Codex CLI still
returns `Access is denied` when invoked directly, so a real authenticated
`codex exec` session and `/hooks` trust review cannot be driven from this
terminal. The repository instead validates the documented hook protocol and
silent/failure-open behavior through reproducible local fixtures and tests.

See [CODEX_COMPATIBILITY.md](CODEX_COMPATIBILITY.md) and
[CODEX_E2E.md](CODEX_E2E.md).

## The replay panel shows no persisted sessions

Replay history begins only after this build has observed and persisted session
data for the selected project. Older sessions from before replay persistence are
not reconstructed.

## Filters make the scene appear empty

This is expected when the current filters remove every visible file. Use
**Reset filters** to restore the full view.

## Windows packaging cannot build the NSIS installer locally

The portable Windows build can be produced locally, but NSIS is not installed
on this host. Windows installer generation is validated by CI. See
[PACKAGING.md](PACKAGING.md) for the packaging boundary.

## Browser or desktop logs show a `THREE.Clock` deprecation warning

The current React Three Fiber dependency path still emits that upstream warning.
It does not indicate corrupted replay or incorrect geometry on its own.
