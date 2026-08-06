# Rendering performance

This document records the reproducible P9-S1 renderer optimization and P9-S2
scale-validation measurements. Results are specific to the environment below;
the deterministic fixtures and commands are retained so another machine can
reproduce the campaign.

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
npm.cmd run benchmark:scale
```

Run the bounded ingress burst benchmark from the repository root:

```powershell
go test ./internal/ingest -run '^$' -bench 'BenchmarkQueueBurst$' -benchmem -count=3
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

Development builds also accept a bounded scale profile. `scale` must be one of
`100`, `1000`, `5000`, `10000`, or `20000`; `history` is clamped from zero to
100,000 exact access records. Diagnostics are enabled automatically unless
`diagnostics=0` is supplied.

```text
http://127.0.0.1:5173/?scale=5000&history=100000
```

The **Add 1,000-event burst** development control drives the ordinary browser
preview bridge one event at a time. The fixtures contain metadata only, are
deterministic and local, and are removed from production behavior by the Vite
development guard.

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

## Scale campaign

The whole-app CPU benchmark creates the deterministic graph and its fixed-size
structure buffer. Long-session rows operate on 100,000 exact trail records and
retain the complete records outside renderer LOD.

| Case | Mean | p99 | Samples |
| --- | ---: | ---: | ---: |
| Build 100-node input | 0.0538 ms | 0.1316 ms | 9,305 |
| Build 1,000-node input | 0.4722 ms | 2.3079 ms | 1,062 |
| Build 5,000-node input | 2.3819 ms | 4.4672 ms | 210 |
| Build 10,000-node input | 4.9673 ms | 9.1469 ms | 101 |
| Build 20,000-node input | 10.6778 ms | 19.9287 ms | 47 |
| Select 100,000-access detail | 78.3127 ms | 91.5882 ms | 10 |
| Calculate 100,000-access analytics | 85.3713 ms | 88.6961 ms | 10 |

The saturated 256-event ingress queue processed offers in 1.898 to 1.954
microseconds per operation with zero benchmark allocations. A deterministic
100,000-event mixed read/patch test completed in 0.416 seconds, never exceeded
capacity, and reported overload drops rather than blocking or growing.

Each browser row is a stable one-second sample after initial load. The
5,000-node row intentionally combines the target interaction scale with the
maximum 100,000-access long-session fixture.

| Nodes | Accesses | FPS | CPU p95 | GPU p95 | Draws | Triangles | Lines | JS heap |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 100 | 200 | 165.0 | 0.20 ms | 0.16 ms | 7 | 42,176 | 310 | 50.7 MB |
| 1,000 | 2,000 | 165.0 | 0.10 ms | 2.00 ms | 7 | 315,536 | 1,266 | 52.8 MB |
| 5,000 | 100,000 | 165.0 | 0.20 ms | 2.14 ms | 7 | 538,416 | 5,266 | 125.2 MB |
| 10,000 | 20,000 | 164.7 | 0.20 ms | 2.16 ms | 7 | 338,416 | 10,266 | 83.6 MB |
| 20,000 | 40,000 | 165.0 | 0.20 ms | 2.21 ms | 7 | 538,416 | 20,266 | 97.8 MB |

At 5,000 nodes and 100,000 accesses, Time-to-Work switching and path-search
result visibility completed in 599 ms and 577 ms respectively, including the
browser-control round trip; the following stable sample returned to 165 fps.
The 1,000-event live browser burst became visible in 289 ms and likewise
returned to 165 fps, seven draw calls, and a 74.6 MB heap sample.

Eight alternating 5,000/20,000-node navigations kept allocations fixed at
seven geometries and one texture. Heap samples were 59.6, 75.9, 70.4, 95.9,
59.6, 84.7, 89.8, and 75.8 MB, showing garbage-collection variation rather
than monotonic growth. Stable frame samples remained between 163.4 and 164.6
fps during this lifecycle run.

The 20,000-node scene degrades geometrical detail and remains responsive, but
the unfiltered shell is visually packed. Individual inspection at that scale
depends on search, filters, focus, and the always-visible selected label rather
than every node being distinguishable at once.

## Validation boundary

These results validate Windows/Chromium on the listed hardware. They are not a
cross-platform GPU guarantee, and JavaScript heap/GPU timing remains
unsupported where the browser omits those APIs. The current renderer maintains
stable interaction around 5,000 visible nodes on this machine and degrades
without freezing at 10,000 and 20,000 nodes. P9-S3 owns security hardening;
P9-S4 owns crash and recovery reliability rather than additional rendering
scale claims.
