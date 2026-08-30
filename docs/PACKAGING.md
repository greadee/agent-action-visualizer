# Packaging

The current release line is defined in `VERSION` and must match
`apps/desktop/wails.json` `info.productVersion`. Release builds inject the
version, commit, and deterministic source timestamp into `internal/buildinfo`;
`aav version` prints that identity as JSON.

## Windows

Windows is the validated packaging target. Install Wails 2.12.0 and optionally
NSIS, then run from the repository root:

```powershell
make package-windows WAILS="C:\path\to\wails.exe"
```

Or call the script directly:

```powershell
.\scripts\package.ps1 -Wails "C:\path\to\wails.exe" -Version 0.1.0
```

The script rejects dirty working trees and mismatched version metadata. Use
`-AllowDirty` only for a non-release local validation build. Use
`-SkipInstaller` when NSIS is unavailable.

The script derives `SOURCE_DATE_EPOCH` from the checked-out commit, uses Wails
`-trimpath`, and writes this complete portable release set to `artifacts/`:

- `agent-action-visualizer-v<version>-windows-amd64.exe`
- `aav.exe`
- `aav-codex-hook.exe`
- `aav-wrapper.exe`
- `SHA256SUMS.txt`

When NSIS is available it also writes
`agent-action-visualizer-v<version>-windows-amd64-installer.exe`. The installer
contains the desktop app and all three setup tools, so the installed application
does not depend on Go, Node.js, npm, Wails, or a source checkout.

The installer is scoped to the current Windows user, installs under
`%LocalAppData%\Programs\greadee\Agent Action Visualizer`, and does not
request administrator elevation. The package workflow runs the following
lifecycle gate before uploading the Windows artifact:

```powershell
.\scripts\test-windows-installer.ps1 `
  -Installer .\artifacts\agent-action-visualizer-v0.1.0-windows-amd64-installer.exe
```

The gate performs install, launch, uninstall, reinstall, a second launch, and a
final uninstall. It verifies the installed desktop executable, companion tools,
shortcuts, current-user uninstall registration, and complete program-file
cleanup. Uninstall deliberately preserves the separate local session journal
and preferences.

For a reproducibility check, rebuild the same clean commit with the same
`-Version`, `SOURCE_DATE_EPOCH`, Go, Node, Wails, and NSIS versions, then compare
the SHA-256 manifest. Signing intentionally changes artifact digests and must
happen after this comparison.

Before publication, a release operator must code-sign every executable and the
installer with a controlled certificate, verify signatures and timestamps,
regenerate the final checksum manifest, and directly exercise install, launch,
uninstall, and reinstall. The August 29, 2026 per-user installer passed that
lifecycle in Windows CI and from the downloaded CI artifact on the validation
host. Code signing has not been implemented, so Windows may display
unknown-publisher or application-reputation warnings for these artifacts.

## Linux and macOS

CI packages Linux/amd64 on Ubuntu and macOS universal on a macOS runner using
native Wails builds. Linux uses Wails' `webkit2_41` tag because current Ubuntu
images provide WebKit 4.1. macOS artifacts are ZIP archives containing the
native `.app` bundle. The project does not claim cross-platform artifacts were
produced from Windows.

Linux requires GCC, GTK3, and WebKit development packages. macOS requires Xcode
Command Line Tools. Run `wails doctor` on the target platform before packaging.
No release is published by CI.
