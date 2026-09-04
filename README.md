# Agent Action Visualizer

Agent Action Visualizer (AAV) is a local-first desktop app that turns activity
inside a code repository into a stable, replayable 3D graph. It helps you see
which files an agent visited, how long it worked there, and the size and
direction of edits without sending repository activity to a model or network
service.

## Start on Windows

Use either release format:

- **Installer:** run
  `agent-action-visualizer-v0.1.0-windows-amd64-installer.exe`. It installs for
  the current user without administrator access and adds Start menu and desktop
  shortcuts.
- **Portable:** extract the complete Windows package and keep
  `agent-action-visualizer-v0.1.0-windows-amd64.exe`, `aav.exe`,
  `aav-codex-hook.exe`, `aav-wrapper.exe`, and `SHA256SUMS.txt` together. Then
  double-click the desktop executable.

Neither format requires Go, Node.js, npm, Wails, a source checkout, or a
terminal to launch the app. Verify downloaded files against `SHA256SUMS.txt`.

Code signing has not been implemented. Windows may show an unknown-publisher
or application-reputation warning for the installer or portable executable.
See the [release checklist](docs/RELEASE_CHECKLIST.md) for validated artifact
names, checksums, and release limitations.

## Open a repository

1. Launch Agent Action Visualizer.
2. Select **Choose folder...** on the first-run screen.
3. Choose a local Git repository in the native Windows folder picker.
4. Wait for the deterministic graph to populate.

The selected repository is remembered locally for quick reopening. You can
switch projects from the app at any time. Invalid or inaccessible paths show a
specific error and a way to choose another folder.

## Read the 3D graph

The repository hierarchy always produces the same layout for the same project
state:

- A sphere represents each visible repository node; file spheres carry
  activity geometry.
- Structure lines connect directories and files.
- An access or edit is anchored at a deterministic pseudo-random point on its
  file sphere, so the same activity does not jump around between renders.
- **Time** mode extrudes positive observed time outward from that point.
- **Work** mode extrudes added lines outward and deleted lines inward from the
  same point.
- Tooltips and the inspector retain exact duration and line counts even when
  long geometry is visually capped.

Use **Access points** and **Activity extrusions** to show or hide those details.
Turning either off reduces geometry and rendering work for large repositories
or long histories. You can also choose linear or logarithmic scaling, adjust
the visual cap, and toggle structure lines, labels, session trails, and local
render diagnostics.

## Explore activity

- **Auto-follow** tracks the active file. Interact with the scene to inspect it
  manually, then use **Return live** or the resume delay to follow activity
  again.
- **Pause updates** freezes the displayed focus and trail without stopping the
  observed command.
- **Session replay** opens persisted sessions and provides timeline and
  playback controls.
- **Filters** narrow the scene by path, file type, operation, confidence,
  agent, or time range. **Reset filters** restores the full view.
- **Analytics** summarizes visible accesses, files, duration, line changes,
  operations, directories, agents, and evidence confidence.

The chosen Time or Work mode is restored when a recent project is reopened.
See the [user guide](docs/USER_GUIDE.md) and
[timeline/replay guide](docs/TIMELINE_REPLAY.md) for every control.

## Connect live activity

Opening a repository works without an agent connection: the app still shows
the static graph and any saved local history. To collect new activity, open the
in-app setup guidance for one of these integrations.

### Codex

The packaged `aav.exe` installs, checks, tests, and removes the local Codex hook.
For a project-local setup, run these commands from the release directory:

```powershell
.\aav.exe codex install --scope project --project C:\path\to\repository
.\aav.exe codex status --scope project --project C:\path\to\repository
.\aav.exe codex test --scope project --project C:\path\to\repository
```

Codex may ask you to trust the project or approve a new hook. The installer
manages only marked AAV hook entries and can remove them with `codex uninstall`.
Read [Codex setup](docs/CODEX_INSTALLER.md) for user-level setup, dry runs, and
recovery behavior.

### Other local agents and commands

Run a command through the packaged generic wrapper:

```powershell
.\aav-wrapper.exe --project-root C:\path\to\repository -- your-agent-command argument
```

The wrapper preserves the child command's arguments, input, output, files, and
exit behavior. Read the [generic wrapper guide](docs/GENERIC_WRAPPER.md) for
flags and integration details.

## Empty, disconnected, and error states

- **First run:** choose a repository to build the graph.
- **Empty history:** the graph remains available, but no activity points or
  extrusions are invented.
- **Disconnected collector:** saved history, graph inspection, filters,
  analytics, and replay remain usable. New visualization evidence may be lost
  until the collector reconnects.
- **Empty filtered view:** reset filters to restore visible files.
- **Invalid or inaccessible folder:** choose another repository or restore
  access, then retry.

Observation is deliberately failure-open. If a hook, wrapper parser, filesystem
watcher, or collector fails, the agent or wrapped command continues with its
normal output, files, and exit status.

## Local data and privacy

AAV stores bounded metadata such as project-relative paths, timestamps,
operations, duration, and line counts in a local journal. It does not persist
source contents, prompts, model responses, command arguments, environment
variables, or transcript contents by default. It does not call a model or send
visualization summaries back into an agent context.

Uninstalling the Windows app removes program files and shortcuts but preserves
the separate local journal and preferences for recovery or reinstallation.
Read [privacy](docs/PRIVACY.md), [troubleshooting](docs/TROUBLESHOOTING.md), and
[known limitations](docs/KNOWN_LIMITATIONS.md) for the full operating boundary.

Agent Action Sync remains deliberately deferred.

## Contributing

Source setup, validation commands, packaging, branch naming, and architecture
references live in [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Licensed under the Apache License 2.0. See [LICENSE](LICENSE).
