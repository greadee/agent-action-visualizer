# Desktop graph bridge

The Wails backend exposes `LoadProject(root)` and `RefreshProject()`. Loading scans project metadata, assigns stable identities, computes the deterministic layout, returns a full snapshot, and emits `aav:graph:snapshot`. Refreshing preserves prior positions, increments the revision, returns a patch, and emits `aav:graph:patch`.

The frontend subscribes to both events. Patches at or below the current revision are ignored. Newer patches update nodes by stable ID and replace the structural edge set, so additions, removals, and moves cannot leave stale edges. A later full snapshot always provides a deterministic resynchronization point after reload or reconnect.

No file contents cross this bridge. The graph DTO contains relative paths, node kinds, stable IDs, coordinates, edges, and a revision only.
