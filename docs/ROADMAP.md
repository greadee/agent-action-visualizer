# Phase and slice roadmap

Status values are maintained in `docs/progress/IMPLEMENTATION_LOG.md`.

| Phase | Slices | Outcome |
| --- | --- | --- |
| P0 | P0-S1 through P0-S3 | Repository, decisions, reproducible toolchain, desktop scaffold, and CI |
| P1 | P1-S1 through P1-S4 | Versioned protocol, bounded ingestion, session core, and SQLite persistence |
| P2 | P2-S1 through P2-S3 | Ignore-aware scanner, stable identity, and deterministic 3D layout |
| P3 | P3-S1 through P3-S3 | Static 3D renderer, focus camera, controls, inspector, and legend |
| P4 | P4-S1 through P4-S3 | Live bridge, active/previous focus, auto-follow, and ordered trail |
| P5 | P5-S1 through P5-S4 | Access anchors, duration/work extrusions, scaling, and geometry validation |
| P6 | P6-S1 through P6-S4 | Verified Codex lifecycle integration, installer, benchmark, and E2E evidence |
| P7 | P7-S1 through P7-S4 | Adapter SDK, wrapper, filesystem fallback, and optional Claude adapter |
| P8 | P8-S1 through P8-S3 | Replay timeline, filters, saved preferences, and deterministic analytics |
| P9 | P9-S1 through P9-S4 | Rendering scale, security, recovery, and cross-platform hardening |
| P10 | P10-S1 through P10-S3 | Packaging, complete documentation, clean-clone validation, and draft PR |

No later phase changes the core rule: visualization remains deterministic, local, asynchronous, and outside model context.
