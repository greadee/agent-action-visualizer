# Work-delta pipeline

Work geometry is calculated locally and asynchronously. Event submission updates focus immediately, marks eligible access intervals as pending, and offers metadata to a bounded queue. A background worker resolves batches without blocking the agent, bridge call, or renderer.

## Evidence order

1. Structured `lines_added` and `lines_deleted` values are exact evidence and take priority.
2. A structured unified patch can be counted in memory when supplied by a future native adapter.
3. Before/after snapshots are correlated only when the caller explicitly enables that already-permitted path. Snapshot bytes are copied into bounded in-memory work, never persisted, and rejected as unknown when the comparison would exceed the calculation bound.
4. Remaining paths are grouped by project root and sent through one batched `git diff --numstat` invocation.
5. Missing rows, unavailable Git, canceled work, or calculation failure produce `unknown`; they never produce invented zeroes.

Binary input is reported as `binary`. Invalid UTF-8 is reported as `unsupported_encoding`. A known `+0 / -0` result is `empty`, which remains distinct from unknown.

## Queue behavior

The desktop queue holds 64 requests and calculates up to 16 together. Offers are non-blocking, pending keys are deduplicated, and overflow fails open with an unknown result. Shutdown cancels Git deadlines and closes workers without changing agent output or files. Git fallback has a two-second local deadline.

## Renderer geometry

Each work result uses the same deterministic surface anchor as its access point:

- additions extend along the outward vector;
- deletions extend against the outward vector;
- mixed results share one anchor;
- exact `+N / -N` values remain available after visual clamping;
- empty, unknown, binary, unsupported-encoding, and pending states use inspectable markers.

Positive work sizes use a minimum visible length, logarithmic scaling by default, and a 1,000-line visual cap. The exact count is never changed by that cap. Linear scaling is also available for the P5-S4 controls.
