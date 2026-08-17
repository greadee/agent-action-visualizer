# Desktop application

This directory contains the Wails desktop shell that hosts the local React and
Three.js frontend for Agent Action Visualizer.

## Development

Install frontend dependencies first:

```powershell
cd frontend
npm ci
cd ..
```

Run the desktop application:

```powershell
wails dev
```

Build a production desktop binary:

```powershell
wails build
```

## Scope

The desktop application owns:

- local collector startup and shutdown
- Wails bindings and event publication
- SQLite-backed replay persistence wiring
- renderer resync and reconnect behavior
- the local React/Three.js experience

Repository-wide contracts and user-facing setup live in the root
[README.md](../../README.md) and the `docs/` directory.
