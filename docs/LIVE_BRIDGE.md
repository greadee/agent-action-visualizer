# Desktop graph bridge

The Wails backend exposes `LoadProject(root)` and `RefreshProject()`. Loading scans project metadata, assigns stable identities, computes the deterministic layout, returns a full snapshot, and emits `aav:graph:snapshot`. Refreshing preserves prior positions, increments the revision, returns a patch, and emits `aav:graph:patch`.

The frontend subscribes to both events. Patches at or below the current revision are ignored. Newer patches update nodes by stable ID and replace the structural edge set, so additions, removals, and moves cannot leave stale edges. A later full snapshot always provides a deterministic resynchronization point after reload or reconnect.

No file contents cross this bridge. The graph DTO contains relative paths, node kinds, stable IDs, coordinates, edges, and a revision only.

## Live focus

Adapters publish validated protocol events through `PublishActivityEvent(event)`. The backend normalizes every primary and secondary path against the loaded project root, maps known paths to stable graph-node IDs, advances the session focus engine, and emits `aav:focus`. Events are rejected when no project is loaded or when any supplied path escapes the selected root.

The focus payload contains only session and activity metadata: current and previous node IDs/paths, secondary node IDs/paths, operation, source, confidence, timestamp, and the ordered access trail. Trail entries carry sequence/timing/operation/source metadata and never source contents. Current focus takes visual precedence over previous focus, which takes precedence over secondary and older-trail focus when roles overlap.

Auto-follow is enabled by default. A search selection, node selection, or manual orbit pauses only camera following for the configured inactivity delay; live focus markers continue to update. **Pause updates** freezes the displayed focus state until **Return live** is selected. Recenter targets the current active file when one is available.

The review-project generator writes `.aav/focus-review-v1.json` with deterministic current, previous, and secondary focus transitions. The development-only **Next review event** control publishes the same event shape through Wails, with an in-browser fallback used only for frontend review.
