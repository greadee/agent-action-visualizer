# Camera focus transitions

Selecting or recentering a node creates a bounded, interruptible camera transition. The root overview uses a distance of 28 graph units so the seven-unit node shell and four-unit activity extrusions fit in frame. File focus is clamped between 8 and 11 units from the selected node and is placed on that node's outward radial axis. It looks back at the selected node, preserving its latest-involvement cluster without changing layout coordinates.

For focus distance `d` in graph units, duration is:

```text
duration_ms = clamp(220 + 55d, 220, 900)
```

Position and orbit target use cubic ease-in-out. Orientation uses Three.js quaternion spherical interpolation (`slerpQuaternions`) over the same normalized progress. Starting any manual orbit, pan, or zoom cancels the active transition. The recenter action starts a new transition. When the operating system requests reduced motion, the final pose is applied immediately.
