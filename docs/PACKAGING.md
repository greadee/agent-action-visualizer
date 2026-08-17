# Packaging

The current release line is defined in `VERSION` and must match
`apps/desktop/wails.json` `info.productVersion`. Release builds inject the
version, commit, and deterministic source timestamp into
`internal/buildinfo`; `aav version` prints that identity as JSON.

## Windows

Windows is the validated packaging target. Install Wails 2.12.0 and NSIS, then
run from the repository root:

```powershell
make package-windows WAILS="C:\path\to\wails.exe"
```

Or call the script directly:

```powershell
.\scripts\package.ps1 -Wails "C:\path\to\wails.exe" -Version 0.1.0
```

The script rejects dirty working trees and mismatched version metadata. Use
`-AllowDirty` only for a non-release local validation build.

The script derives `SOURCE_DATE_EPOCH` from the checked-out commit unless it is
provided explicitly, uses Wails `-trimpath`, and writes these ignored artifacts
to `artifacts/`:

- `agent-action-visualizer-v<version>-windows-amd64.exe`
- `agent-action-visualizer-v<version>-windows-amd64-installer.exe`
- `SHA256SUMS.txt`

For a reproducibility check, rebuild the same clean commit with the same
`-Version`, `SOURCE_DATE_EPOCH`, Go, Node, Wails, and NSIS versions, then compare
the generated SHA-256 manifest. The build recipe is reproducible; code signing
will intentionally change an artifact digest and must happen after this check.

## Linux and macOS

CI packages Linux/amd64 on Ubuntu and macOS universal on a macOS runner using
native Wails builds. Linux uses Wails' `webkit2_41` tag because current Ubuntu
images provide WebKit 4.1. macOS artifacts are ZIP archives containing the
native `.app` bundle. The project does not claim cross-platform artifacts were
produced from Windows.

Linux requires GCC, GTK3, and WebKit development packages. macOS requires Xcode
Command Line Tools. Run `wails doctor` on the target platform before packaging.

Artifacts are not signed, macOS artifacts are not notarized, and no release is
published by CI. Signing and notarization require controlled credentials and
are release-operator steps.
