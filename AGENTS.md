# Repository instructions

- Keep observation deterministic and local. Do not add AI/model calls or model-visible reporting for visualization.
- Preserve agent behavior, output, exit status, and working files. Adapters must fail open.
- Work in the phase/slice sequence recorded in `docs/progress/IMPLEMENTATION_LOG.md`.
- Run relevant checks before each commit and push each completed slice before starting the next.
- Prefer commit subjects beginning with lowercase action words such as `add`, `upd`, or `fix` and include the phase/slice ID.
- Never commit source contents, credentials, local databases, generated binaries, dependency directories, or temporary snapshots.
