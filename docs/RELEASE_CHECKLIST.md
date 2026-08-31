# Windows initial-release validation

This checklist records Phase 11 validation performed on August 26, 2026. The
clean-clone package was built from implementation commit
`89b31736e450af73e46efcef4bc91659ffeb24ef`; the documentation record follows
in a separate commit.

## Validation host

- Windows `10.0.26200.9168`, Windows/amd64
- AMD Ryzen 5 9600X, 12 logical processors
- 33,460,023,296 bytes physical memory
- NVIDIA GeForce RTX 3080, 10,240 MiB, driver `610.47`
- Go `1.26.6`, Node.js `22.22.3`, npm `10.9.8`, Wails `2.12.0`, and
  Git `2.50.1.windows.1`
- NSIS: unavailable on the host (`makensis.exe` was absent from `PATH` and the
  standard Program Files locations)

## Commands and results

Repository and frontend gates:

```powershell
git diff --check
go mod verify
go test ./...
go vet ./...
cd apps/desktop
go mod verify
go test ./...
go vet ./...
cd frontend
npm.cmd run format:check
npm.cmd run lint
npm.cmd run typecheck
npm.cmd test -- --run
npm.cmd run build
```

All passed. The frontend suite contained 20 files and 64 tests; the focused
activity/onboarding suite contained 5 files and 27 tests. Existing replay,
filters, analytics, Time, and Work fixtures passed with the complete suite.

Windows production build:

```powershell
cd apps/desktop
& "C:\Program Files\Wails\bin\wails.exe" build -clean -platform windows/amd64 -trimpath
```

Passed. The embedded frontend, Go desktop module, Windows resources, and
production executable built successfully.

Clean-clone package build:

```powershell
$clone = Join-Path (Resolve-Path .cache) "p11-final-clean-89b3173"
$artifacts = Join-Path (Resolve-Path .cache) "p11-final-artifacts-89b3173"
git clone --no-local . $clone
git -C $clone checkout 89b31736e450af73e46efcef4bc91659ffeb24ef
& "$clone\scripts\package.ps1" -SkipInstaller -OutputDirectory $artifacts `
  -Wails "C:\Program Files\Wails\bin\wails.exe"
git -C $clone status --short --branch
Get-ChildItem $artifacts -File | Get-FileHash -Algorithm SHA256
```

Passed. The clean clone remained clean after the build; only ignored build
outputs were created. `aav.exe version` reported version `0.1.0`, commit
`89b31736e450`, and build date `1787796038`. All four executables reported
`NotSigned` through Windows Authenticode inspection.

Tracked-content and generated-output audit:

```powershell
git ls-files | Select-String -Pattern `
  '(node_modules|(^|/)(artifacts|dist|bin|package-tools)/|\.(db|sqlite|exe|dll)$)'
git ls-files | Select-String -Pattern '(snapshot|source-snapshot)'
$profilePattern = [regex]::Escape($env:USERPROFILE)
git grep -n -I -E '(BEGIN (RSA |OPENSSH )?PRIVATE KEY|AKIA[0-9A-Z]{16})'
git grep -n -I -- $profilePattern
git check-ignore -v apps/desktop/build/windows/installer/package-tools/aav.exe
git diff --check
```

Passed. No generated binaries, databases, dependency directories, source
snapshots, credentials, secrets, or host-specific user paths are tracked. The
generated package-helper directory is ignored.

## Artifact record

The exact portable release set built locally from the clean clone is:

| Artifact                                           |      Bytes | SHA-256                                                            |
| -------------------------------------------------- | ---------: | ------------------------------------------------------------------ |
| `agent-action-visualizer-v0.1.0-windows-amd64.exe` | 16,845,312 | `2f8cb2ff56b205545dc21f9a74e2d925a81f3d4b60d9fec231384ccecc668802` |
| `aav.exe`                                          |  4,583,936 | `9b12ed620c2fb647a10e396f6291c7a7374b9f4e4e34c8f3cb30e56289bf95ff` |
| `aav-codex-hook.exe`                               |  4,150,272 | `dd800b888c5d7350d1a5f746dfbd8b4b768d7c4207d0f0b0a7e68de28051429d` |
| `aav-wrapper.exe`                                  |  5,137,920 | `17e672d075cd07223e705ac300ecdcf9373c74a42f8cbb8e02f01c2946d9ddc0` |
| `SHA256SUMS.txt`                                   |        360 | `e0214bd728e72bfb7ad9ac84fe6d3a9f49c60e5cc4e52000178666bc28f5b32c` |

Windows package workflow run
[`33031924687`](https://github.com/greadee/agent-action-visualizer/actions/runs/33031924687)
also completed successfully at the same implementation commit. Its downloaded
artifact `agent-action-visualizer-v0.1.0-windows-amd64` contained:

| Artifact                                                     |      Bytes | SHA-256                                                            |
| ------------------------------------------------------------ | ---------: | ------------------------------------------------------------------ |
| `agent-action-visualizer-v0.1.0-windows-amd64.exe`           | 16,845,312 | `76914ef3767759753a769943d3f7da21f8dc013aaa09da572fbfab96f5cdf26f` |
| `agent-action-visualizer-v0.1.0-windows-amd64-installer.exe` | 16,382,590 | `c4877b196264c3003068cda8a3d2145dc5f5aebb8288846f67ee183bec878ea4` |
| `aav.exe`                                                    |  4,583,936 | `ec5a5c8b9416fea6386537582743a60cc0ce1d10b65a5f7efdb0d442195ea66c` |
| `aav-codex-hook.exe`                                         |  4,150,272 | `1f3a7ba2be5cef5615685cf86b472c441587cf33a6a07601d4af7a87e4e794fc` |
| `aav-wrapper.exe`                                            |  5,137,920 | `1c852a137475717b570387e22c328de99761f3c1aebc8225cfd949fe73e07084` |
| `SHA256SUMS.txt`                                             |        486 | `720e9d4a7c1dea961b4517901fe584fc542c43960874c0fe0b0e5706c75c6cf7` |

The workflow manifest matched all five executable hashes. Authenticode reported
`NotSigned` for every executable.

## Desktop and visual validation

The packaged executable was launched directly through Windows Explorer without
Go, Node.js, npm, Wails, or a terminal participating in the launch. The review
covered:

- first run and the native **Choose a local Git repository** dialog;
- valid populated repository graph;
- invalid/inaccessible path alert;
- empty history and disconnected collector states;
- Codex setup, generic-wrapper setup, help, and troubleshooting guidance;
- close/reopen persistence of the recent project and Time/Work mode;
- local-only static graph availability while the collector was disconnected;
- labels, control names, grouping, tab order, keyboard activation, and
  empty/error status announcements; and
- replay, filters, analytics, Time, and Work behavior through both browser
  fixtures and the complete automated suite.

The activity-geometry review used a 100-node/40-access deterministic fixture.
Time mode showed orange positive-duration extrusions. Work mode showed teal
addition and pink deletion extrusions from the same stable per-file points.
Disabling **Activity extrusions** removed the lines while retaining points;
disabling **Access points** removed the point layer. Analytics reported 40
accesses, 40 files, 13m10s, and +780/-780. The only console message was the
known upstream `THREE.Clock` deprecation warning.

The direct desktop run used the application code at `6c297dc8`, immediately
before the packaging-only companion-tool and workflow change at `89b31736`.
A repeat launch of the exact `89b31736` package was blocked by an unrelated
Windows Security firewall prompt owned by `api.test`; automation policy did not
permit interacting with that prompt. The exact final package was nevertheless
built from a clean clone, version-inspected, checksummed, and exercised through
the same complete application test/build gates. This is the remaining risk in
claiming byte-for-byte final executable UI validation.

The native picker opened and was inspected, but its owned Explorer child
controls were not addressable by the desktop automation bridge. Native folder
selection is covered by Wails binding integration tests; completing a folder
choice inside the real dialog was not automated.

## Unavailable release gates and remaining risk

- The portable release set was built and the preceding identical application
  code was directly exercised without developer tooling. The exact-final repeat
  limitation is recorded above.
- NSIS remains unavailable locally, so installer compilation is a Windows CI
  responsibility. The final per-user installer passed two complete
  install/launch/uninstall cycles in CI and again after download on this Windows
  host. The exact lifecycle evidence is recorded in the follow-up below.
- Code signing has not been implemented. Windows may display an
  unknown-publisher or application-reputation warning.
- A real authenticated Codex turn remains unavailable because the installed
  AppX Codex CLI returns `Access is denied`; deterministic hook installer and
  failure-open contracts pass their local fixtures.
- Linux and macOS artifacts remain CI-only and outside this Windows release
  validation.
- Agent Action Sync remains deliberately deferred. No model calls or
  model-visible visualization reporting were added.

## Per-user installer follow-up

The August 29, 2026 P12-S1 follow-up resolved the Phase 11 installer lifecycle
limitation. Implementation commits `12f6e8a010e5862c18cb97245e50014a9e2527e2`
and `d03c9c34b1dd99ba342a6fe8dd5c201104e7ea43` changed NSIS from an
administrator-scoped Program Files install to a current-user install under
`%LocalAppData%\Programs\greadee\Agent Action Visualizer`.

The installer now:

- avoids UAC and writes uninstall metadata under HKCU;
- installs `agent-action-visualizer.exe`, `aav.exe`,
  `aav-codex-hook.exe`, and `aav-wrapper.exe`;
- creates current-user Start menu and desktop shortcuts;
- removes program files, shortcuts, and registration on uninstall; and
- preserves the separate local journal and WebView preferences.

CI run
[`33274950901`](https://github.com/greadee/agent-action-visualizer/actions/runs/33274950901)
and package run
[`33274950955`](https://github.com/greadee/agent-action-visualizer/actions/runs/33274950955)
both passed at `d03c9c34`. The Windows job compiled NSIS 3.12 and reported:

```text
cycle=1 action=install
cycle=1 action=launch
cycle=1 action=uninstall
cycle=2 action=install
cycle=2 action=launch
cycle=2 action=uninstall
installer_lifecycle=passed
```

The exact downloaded Windows artifact then passed the same two cycles directly
on the validation host. Final checks confirmed no install directory, uninstall
registration, shortcut, or application process remained, while the local
journal and WebView data remained present.

| Artifact                                                     |      Bytes | SHA-256                                                            |
| ------------------------------------------------------------ | ---------: | ------------------------------------------------------------------ |
| `agent-action-visualizer-v0.1.0-windows-amd64.exe`           | 16,845,312 | `e1ff4dc800548621a510995e8a842b318f132f7ec8aa8b01fe8abf776147b369` |
| `agent-action-visualizer-v0.1.0-windows-amd64-installer.exe` | 16,382,630 | `93d27667950127a81a8b83930c7838101696a303726a9b3ad2c1768b9692e852` |
| `aav.exe`                                                    |  4,583,936 | `0ae6b6c650ef7c7da42d122dcad576fe685b4bd9387e2c39e95eaf20a3625cb7` |
| `aav-codex-hook.exe`                                         |  4,150,272 | `ea10b6007f64d731253474838ec35e9610473514a2d00a476b9bd86cce6f15eb` |
| `aav-wrapper.exe`                                            |  5,137,920 | `c3c0cd7c346b2469ed6d80309acdd9eb72da57d04efb1f88c0d119ce28ea8ed5` |
| `SHA256SUMS.txt`                                             |        486 | `0643e4a61217ca2af0c1c140a157dd5779fb71070ad3d7a0f250d4dde38999b2` |

The manifest matched all five executable hashes. Code signing has not been
implemented, so all executables remain unsigned and Windows may display
unknown-publisher or application-reputation warnings.

## GitHub Actions allocation follow-up

On August 30, 2026, pull request 20 created CI run `33327495227` and package
run `33327495223`. Three attempts of each run failed before checkout. Every Go,
frontend, Windows, Linux, and macOS job contained no steps and produced no job
log; the dependent desktop job was skipped. This is an account-level hosted
runner allocation failure rather than a repository test result.

The P13-S1 follow-up removes duplicate feature-push/PR validation, consolidates
routine checks onto one Ubuntu job, restricts native packaging to `v*` tags or
manual requests, cancels superseded runs, and bounds job duration. Restoring an
already-exhausted private-repository runner allowance still requires quota
renewal, an Actions budget change, or a secured self-hosted runner.
