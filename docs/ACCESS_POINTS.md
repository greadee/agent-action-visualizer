# Access points

P5-S1 renders one point per normal session access interval from the metadata-only `aav:focus` trail. A point uses the stable graph node ID, relative-path event metadata, sequence, operation, timestamp, and source confidence; file contents never enter the renderer.

## Deterministic anchors

Each node owns a local ordinal sequence ordered by access sequence. Ordinal zero sits at the outward-facing center of the node. Later ordinals use a golden-angle Fibonacci disk on that outward-facing hemisphere. The anchor is offset `0.29` graph units from the node center, so it stays inspectable as a point on the node surface and does not change after later accesses arrive.

## Dense history

Nodes with 24 or fewer accesses render one point per interval. Above that threshold, the newest 16 stay individual and the older accesses are compacted into at most eight deterministic bucket points. An aggregated point retains the count and inclusive sequence range it represents; hover metadata reports the latest access within that bucket. This bounds a dense node to 24 rendered point instances while preserving the complete access count.

Individual closed points are pale blue, the currently open interval is teal, and aggregated points are purple. Access-point visibility is independent of project structure edges, session trails, and the pre-existing report-based activity extrusions.

The review-project generator writes `.aav/access-points-review-v1.json`, a deterministic 60-access fixture alternating two files so each crosses the aggregation threshold.
