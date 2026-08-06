# Rendering performance

This document records the reproducible P9-S1 renderer measurements. It does
not replace the larger node-count, burst, memory-growth, and long-session
campaign assigned to P9-S2.

## Environment

- Date: 2026-08-05
- OS: Windows NT 10.0.26200.0, amd64
- CPU: AMD Ryzen 5 9600X, 12 logical processors
- GPU: NVIDIA GeForce RTX 3080; AMD Radeon Graphics also installed
- Browser: Codex in-app Chromium browser at a 165 Hz display cadence
- Runtime: Node.js 22.22.3, npm 10.9.8, Go 1.26.3
- Frontend: React 19.2.8, Three.js 0.185.1, React Three Fiber 9.6.1

The browser exposed `EXT_disjoint_timer_query_webgl2`, so GPU p95 values are
measured timer-query results. JavaScript heap values use Chromium's optional
`performance.memory` surface. Both fields display `unsupported` when a runtime
does not expose them.

## Commands

Run the deterministic render-model benchmark from
`apps/desktop/frontend`:

```powershell
npm.cmd run benchmark:render
```

Build and inspect production chunk sizes with:

```powershell
npm.cmd run build
```

For live measurements, run `npm.cmd run dev`, enable **Render diagnostics**,
and hold each representative state for at least one complete one-second sample
window. The panel measures CPU time around the renderer submission, GPU elapsed
time where timer queries are supported, frame cadence, draw calls, primitives,
WebGL allocations, and JavaScript heap use. Diagnostics remain local and are
disabled by default.

## Bundle result

| Build | Entry JS | Entry gzip | Largest chunk | Result |
| --- | ---: | ---: | ---: | --- |
| P8-S3 baseline | 1,150.40 kB | 316.85 kB | 1,150.40 kB | One chunk; Vite warning |
| P9-S1 | 233.30 kB | 72.66 kB | 373.73 kB | Renderer split; no size warning |

The application entry is 79.7% smaller uncompressed and 77.1% smaller gzip.
The first visible scene still requests the deferred renderer immediately, so
this primarily allows the HTML controls and loading state to paint before
Three.js initialization; it does not claim a reduction in the complete
renderer payload.

## Render-model benchmark

Vitest 4.1.10 benchmark results from one reproducible run:

| Case | Mean | p99 | Samples |
| --- | ---: | ---: | ---: |
| Select 1,000 activity accesses | 0.0619 ms | 0.1006 ms | 8,078 |
| Select 10,000 activity accesses | 0.6185 ms | 1.1695 ms | 809 |
| Select 10,000 access points | 1.5286 ms | 2.2736 ms | 328 |

These measure deterministic CPU-side LOD selection only, not graph layout,
React reconciliation, browser paint, or GPU rendering.

## Browser measurements

Each row is a stable one-second panel sample at the default camera. The dense
fixture is five deterministic 60-access batches; it renders 256 detailed
activity accesses from 300 exact records and retains active, inspected, and
scale-defining evidence.

| State | FPS | CPU p95 | GPU p95 | Draw calls | Triangles | Lines | Geometries | JS heap |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Base, 8 nodes | 166.2 | 0.10 ms | 0.04 ms | 2 | 1,440 | 7 | 2 | 52.9 MB |
| Dense time, 300 accesses | 165.0 | 0.20 ms | 0.14 ms | 7 | 25,616 | 274 | 7 | 54.0 MB |
| Mixed work, 7 accesses | 165.0 | 0.20 ms | 0.06 ms | 8 | 4,488 | 19 | 8 | 62.8 MB |

The frame rate is display-cadence limited in these small fixtures. A clean
browser run produced no errors. React Three Fiber still emits the known
`THREE.Clock` deprecation warning from its current dependency path.

## LOD and lifecycle guarantees

- Nodes use one instanced mesh and step icosahedron detail down above 1,000 and
  5,000 visible nodes.
- Activity detail is bounded at 256 accesses. Active, inspected, maximum
  duration, maximum addition, and maximum deletion records are retained before
  deterministic recent/older sampling.
- Access points retain their established per-file aggregation and are globally
  bounded at 2,048 instances, preserving active, inspected, aggregate, and
  recent points.
- Endpoint sphere tessellation steps down above 48 and 128 instances.
- Selected labels always remain visible; hover-label range tightens above
  1,000 and 5,000 nodes.
- Structure, trail, and activity lines use buffer geometry. Replaced buffers
  are explicitly disposed.
- Diagnostics do not alter persisted evidence, replay, analytics, adapters, or
  model context.

## Deferred validation

P9-S2 owns the complete 100/1,000/5,000/10,000/20,000-node campaign,
high-frequency burst tests, sustained memory-growth checks, long-session tests,
and hardware-dependent graceful-degradation limits. P9-S1 establishes the
instrumentation and bounded policies needed for that work without claiming
those scale gates are complete.
