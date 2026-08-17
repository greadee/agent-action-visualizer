# Timeline and replay

Replay reconstructs persisted local session history without changing the live
collector state. The same normalized records produce the same graph, focus,
trail, and activity geometry.

## Session selection

Each loaded project exposes a list of persisted sessions in the **Session
replay** panel.

- Selecting **Live session** exits replay and returns to the live collector.
- Selecting a persisted session opens its latest event by default.
- If no sessions exist for the project, the panel reports that explicitly.

Replay history begins only after this application version has written journal
data for the selected project.

## Timeline controls

Replay supports:

- a range-slider timeline over persisted event positions
- **Previous access** and **Next access** navigation
- **Play** and **Pause**
- playback speeds of `0.5x`, `1x`, `2x`, and `4x`
- **Return live** to leave replay mode

The replay cursor operates on persisted event order. Previous/next access jumps
between access-bearing cursor positions rather than every stored event.

## What replay reconstructs

Replay restores:

- graph snapshot and patch order
- active and previous focus
- session trail
- access points
- Time extrusions
- Work additions and deletions
- persisted exact duration and work values

Replay and live state are kept separate in the frontend. Opening replay freezes
the displayed live state until **Return live** is chosen.

## Determinism and ordering

Persisted replay uses canonical ordering over session lifecycle, timestamps,
optional monotonic timestamps, and event IDs. Duplicate event IDs are ignored.
Late live events cannot move replay backward or roll focus behind the latest
accepted live state.

## Filter and analytics behavior

Replay uses the replay cursor time when evaluating relative time filters and
analytics. Live mode uses current wall-clock time instead. This keeps filtered
views and analytics deterministic for a chosen replay position.

## Known replay boundaries

- Replay depends on the local SQLite journal remaining available.
- Sessions from before replay persistence was added cannot be reconstructed.
- If a corrupt record is encountered, replay fails closed for that session
  rather than presenting partial fabricated history.
