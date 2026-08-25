# Codex hook installer

P6-S3 adds reversible project-local and user-level installation for the
`aav-codex-hook` executable. The installer uses Codex's documented
`hooks.json` source and does not edit `config.toml`, prompts, model settings,
environment configuration, or managed policy.

## Build and install

Build the CLI and hook beside one another:

```powershell
New-Item -ItemType Directory -Force .cache\aav-bin | Out-Null
go build -o .cache\aav-bin\aav.exe ./cmd/aav
go build -o .cache\aav-bin\aav-codex-hook.exe ./cmd/aav-codex-hook
```

Install for one project:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope project --project C:\path\to\repository
```

Install for the current Codex user:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope user
```

`--codex-home C:\path\to\.codex` overrides `CODEX_HOME` for an isolated
user-level installation. `--hook-binary` overrides the adjacent
`aav-codex-hook.exe` lookup. Paths with spaces are supported in both
`command` and `commandWindows`.

Preview either mutation without writing:

```powershell
.\.cache\aav-bin\aav.exe codex install --scope project --project C:\path\to\repository --dry-run
.\.cache\aav-bin\aav.exe codex uninstall --scope project --project C:\path\to\repository --dry-run
```

## Status, diagnostic, and removal

```powershell
.\.cache\aav-bin\aav.exe codex status --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex test --scope project --project C:\path\to\repository
.\.cache\aav-bin\aav.exe codex uninstall --scope project --project C:\path\to\repository
```

`status` reports `not-installed`, `installed`, `partial`, or `invalid`.
`test` starts an isolated local collector, invokes the installed hook with a
sanitized `SessionStart` payload, and requires one valid event, exit code zero,
and empty stdout/stderr. It does not invoke Codex or any model.

Codex separately reviews non-managed hooks. A project-level hook is loaded
only when Codex trusts the project, and a new or changed hook command may
require approval through Codex's `/hooks` review. Filesystem status and the
isolated diagnostic deliberately do not claim that approval has happened.
P6-S4 owns trusted live-session validation.

## Mutation and recovery contract

- Project scope owns marked handlers in `<project>\.codex\hooks.json` and a
  managed binary below `<project>\.codex\aav\bin`.
- User scope owns the same files below `CODEX_HOME` or `~\.codex`.
- The first install saves the exact existing `hooks.json` bytes and SHA-256
  before configuration mutation. Repeated installs retain that first backup.
- The JSON merger removes or replaces only handlers carrying both the AAV
  status label and installation marker. Unrelated root keys, hook events,
  groups, and handlers are retained.
- Uninstall restores the exact backup when no unrelated semantic change has
  occurred. If configuration changed after installation, uninstall removes
  only AAV handlers and retains the current unrelated values.
- Invalid JSON, an unexpected configuration shape, a mismatched state file,
  a missing/tampered backup, or an unmanaged destination binary stops
  mutation.
- Same-directory temporary files and swap recovery protect configuration,
  state, backups, and the managed executable from interrupted replacement.
- Status verifies the managed executable digest and the exact six-event hook
  layout recorded by the installer, not merely the presence of those files.
- Diagnostics report action, scope, status, paths, counts, and bounded error
  text. They never print hook configuration values, environment variables, or
  source contents.

If the same configuration layer also defines inline `[hooks]` in
`config.toml`, Codex merges the two sources and warns. The installer does not
rewrite or interpret that unrelated TOML; use one representation per layer
where practical.
