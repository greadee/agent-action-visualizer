# Session trail

The session trail visualizes ordered access intervals without changing project layout or mixing travel with project-structure edges. Every `aav:focus` payload includes the session engine's metadata-only access list: sequence, stable node ID, relative path, start/end timestamps, duration, operation names, source/confidence, and optional agent ID. Source-file contents and command output never cross this bridge.

## Ordering and visibility

Accesses are sorted by their monotonic session sequence. Consecutive accesses to the same logical node may be collapsed before display. Recent mode then selects the last `N` accesses, where `N` is clamped to at least two so a transition can be drawn. Complete-session mode bypasses the recency limit. Missing graph nodes are omitted from rendered transitions rather than connected to an invented position.

Current, previous, and secondary-active node styles retain priority. Other nodes in the visible trail receive a quieter older-access treatment. Pausing live updates freezes the displayed trail together with focus; returning live applies the latest complete focus payload.

## Rendering

Project structure remains a subdued blue-gray line layer. Session travel uses a separate vertex-colored line buffer that brightens toward the newest transition, plus instanced cone markers oriented from the source node toward the destination node. Hovering a direction marker shows the destination access sequence, operation, timestamp, source, and confidence.

The frontend updates existing buffer/instance geometry and does not relayout the graph. The trail visibility toggle is independent from both structure edges and activity extrusions.

The replay cursor and timeline transport remain scheduled for Phase 8. Activity anchors and per-access time/work geometry remain Phase 5 work and are not introduced by P4-S3.
