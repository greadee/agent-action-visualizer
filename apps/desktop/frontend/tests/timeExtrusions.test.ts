import { describe, expect, it } from 'vitest'
import {
  ACTIVE_INTERVAL_CAP_MS,
  MAX_DURATION_EXTRUSION,
  MIN_DURATION_EXTRUSION,
  buildTimeExtrusions,
  durationExtrusionLength,
  durationScaleRatio,
  normalizedIntervalDuration,
} from '../src/activity/timeExtrusions'
import type { GraphNode, TrailAccess, Vec3 } from '../src/graph/types'

const startedAt = Date.parse('2026-07-23T12:00:00.000Z')
const node: GraphNode = {
  id: 'file-a',
  path: 'src/a.ts',
  kind: 'source',
  position: [4, 3, 5],
}

function access(
  sequence: number,
  durationMs: number,
  ended = true,
): TrailAccess {
  return {
    sequence,
    node_id: node.id,
    path: node.path,
    started_at: new Date(startedAt).toISOString(),
    ended_at: ended
      ? new Date(startedAt + durationMs).toISOString()
      : undefined,
    duration_ms: durationMs,
    operations: ['patch'],
    source: 'synthetic',
    confidence: 'exact',
  }
}

function dot(left: Vec3, right: Vec3): number {
  return left[0] * right[0] + left[1] * right[1] + left[2] * right[2]
}

describe('time extrusions', () => {
  it('builds one deterministic outward extrusion from each access anchor', () => {
    const extrusions = buildTimeExtrusions(
      [access(2, 6_000), access(1, 500)],
      [node],
      startedAt + 10_000,
    )
    expect(extrusions.map((extrusion) => extrusion.sequence)).toEqual([1, 2])
    expect(extrusions[0]?.start).not.toEqual(extrusions[1]?.start)
    for (const extrusion of extrusions) {
      const direction: Vec3 = [
        extrusion.end[0] - extrusion.start[0],
        extrusion.end[1] - extrusion.start[1],
        extrusion.end[2] - extrusion.start[2],
      ]
      expect(dot(direction, extrusion.outward)).toBeGreaterThan(0)
    }
  })

  it('uses stable linear and logarithmic duration scaling with visible bounds', () => {
    expect(durationScaleRatio(5_000, 10_000, 'linear')).toBeCloseTo(0.5)
    expect(durationScaleRatio(5_000, 10_000, 'log')).toBeGreaterThan(0.5)
    expect(durationExtrusionLength(1, 10_000, 'log')).toBeGreaterThanOrEqual(
      MIN_DURATION_EXTRUSION,
    )
    expect(durationExtrusionLength(10_000, 10_000, 'linear')).toBe(
      MAX_DURATION_EXTRUSION,
    )
  })

  it('grows a live interval without including idle time beyond the normalized cap', () => {
    const active = access(1, 0, false)
    expect(normalizedIntervalDuration(active, startedAt + 1_250)).toBe(1_250)
    expect(
      normalizedIntervalDuration(
        active,
        startedAt + ACTIVE_INTERVAL_CAP_MS * 3,
      ),
    ).toBe(ACTIVE_INTERVAL_CAP_MS)
  })

  it('retains exact duration values while visually clamping extreme intervals', () => {
    const extrusions = buildTimeExtrusions(
      [access(1, 1), access(2, ACTIVE_INTERVAL_CAP_MS)],
      [node],
      startedAt + ACTIVE_INTERVAL_CAP_MS,
    )
    expect(extrusions[0]?.durationMs).toBe(1)
    expect(extrusions[0]?.length).toBeGreaterThanOrEqual(MIN_DURATION_EXTRUSION)
    expect(extrusions[1]?.durationMs).toBe(ACTIVE_INTERVAL_CAP_MS)
    expect(extrusions[1]?.length).toBe(MAX_DURATION_EXTRUSION)
    expect(extrusions[1]?.clamped).toBe(true)
  })

  it('handles a degenerate node position with a finite outward direction', () => {
    const origin = { ...node, position: [0, 0, 0] as Vec3 }
    const extrusion = buildTimeExtrusions(
      [access(1, 2_000)],
      [origin],
      startedAt,
    )[0]!
    expect(extrusion.outward.every(Number.isFinite)).toBe(true)
    expect(extrusion.end.every(Number.isFinite)).toBe(true)
  })
})
