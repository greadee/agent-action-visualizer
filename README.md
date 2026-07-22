# Agent Action Visualizer

Agent Action Visualizer (AAV) is a local-first desktop application for observing coding-agent activity as a stable, replayable 3D project graph.

The project is under active construction. Its core rule is non-negotiable: adapters observe deterministic local lifecycle, tool, filesystem, and Git events without asking an AI model to narrate its work and without making additional model or API calls.

See [the implementation log](docs/progress/IMPLEMENTATION_LOG.md) for current status.

## Review activity geometry

Create a deterministic Git repository with agent-reported time/work data:

```powershell
go run ./cmd/aav-review-project -out C:\tmp\aav-extrusion-review
```

Load that path in the desktop app to review the uniform node shell, latest-commit grouping, and Time/Work edit-point extrusions. See [the activity geometry contract](docs/ACTIVITY_GEOMETRY.md) for data authority and scaling details.

## License

Licensed under Apache License 2.0. See [LICENSE](LICENSE).
