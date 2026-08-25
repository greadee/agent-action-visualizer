import { describe, expect, it } from 'vitest'
import {
  MAX_WORK_EXTRUSION,
  MIN_WORK_EXTRUSION,
  buildWorkGeometry,
  workExtrusionLength,
} from '../src/activity/workExtrusions'
import type { GraphNode, TrailAccess, Vec3 } from '../src/graph/types'

const node: GraphNode = {
  id: 'file-a',
  path: 'src/a.ts',
  kind: 'source',
  position: [4, 3, 5],
}

function access(
  sequence: number,
  options: Partial<TrailAccess> = {},
): TrailAccess {
  return {
    sequence,
    node_id: node.id,
    path: node.path,
    started_at: '2026-07-24T12:00:00Z',
    ended_at: '2026-07-24T12:00:01Z',
    duration_ms: 1_000,
    operations: ['patch'],
    source: 'synthetic',
    confidence: 'exact',
    work_source: 'structured_patch',
    work_confidence: 'exact',
    ...options,
  }
}

function dot(left: Vec3, right: Vec3): number {
  return left[0] * right[0] + left[1] * right[1] + left[2] * right[2]
}

describe('work extrusions', () => {
  it('projects additions outward and deletions inward from the same anchor', () => {
    const geometry = buildWorkGeometry(
      [
        access(1, {
          lines_added: 25,
          lines_deleted: 9,
          work_status: 'known',
        }),
      ],
      [node],
    )
    expect(geometry.segments).toHaveLength(2)
    expect(geometry.segments[0]?.start).toEqual(geometry.segments[1]?.start)
    const addition = geometry.segments.find(
      (segment) => segment.direction === 'addition',
    )!
    const deletion = geometry.segments.find(
      (segment) => segment.direction === 'deletion',
    )!
    const additionDirection: Vec3 = [
      addition.end[0] - addition.start[0],
      addition.end[1] - addition.start[1],
      addition.end[2] - addition.start[2],
    ]
    const deletionDirection: Vec3 = [
      deletion.end[0] - deletion.start[0],
      deletion.end[1] - deletion.start[1],
      deletion.end[2] - deletion.start[2],
    ]
    expect(dot(additionDirection, addition.outward)).toBeGreaterThan(0)
    expect(dot(deletionDirection, deletion.outward)).toBeLessThan(0)
    expect(addition.value).toBe(25)
    expect(deletion.value).toBe(9)
  })

  it('uses bounded linear and logarithmic scaling without changing exact values', () => {
    expect(workExtrusionLength(1, 100, 'log')).toBeGreaterThanOrEqual(
      MIN_WORK_EXTRUSION,
    )
    expect(workExtrusionLength(50, 100, 'log')).toBeGreaterThan(
      workExtrusionLength(50, 100, 'linear'),
    )
    expect(workExtrusionLength(100, 100, 'linear')).toBe(MAX_WORK_EXTRUSION)
    const segment = buildWorkGeometry(
      [
        access(1, {
          lines_added: 1_000_000,
          lines_deleted: 0,
          work_status: 'known',
        }),
      ],
      [node],
    ).segments[0]!
    expect(segment.value).toBe(1_000_000)
    expect(segment.length).toBe(MAX_WORK_EXTRUSION)
    expect(segment.clamped).toBe(true)
  })

  it('uses the selected visual cap without changing exact replay values', () => {
    const trail = [
      access(1, { lines_added: 100, lines_deleted: 0, work_status: 'known' }),
      access(2, {
        lines_added: 10_000,
        lines_deleted: 0,
        work_status: 'known',
      }),
    ]
    const first = buildWorkGeometry(trail, [node], 'linear', 1_000)
    const replay = buildWorkGeometry(trail, [node], 'linear', 1_000)
    const wide = buildWorkGeometry(trail, [node], 'linear', 10_000)

    expect(first).toEqual(replay)
    expect(first.segments.map((segment) => segment.value)).toEqual([
      100, 10_000,
    ])
    expect(first.segments[1]?.clamped).toBe(true)
    expect(wide.segments[1]?.clamped).toBe(false)
    expect(first.segments[0]?.length).toBeGreaterThan(
      wide.segments[0]?.length ?? 0,
    )
  })

  it('creates inspectable markers for empty, unknown, binary, and unsupported work', () => {
    const geometry = buildWorkGeometry(
      [
        access(1, { lines_added: 0, lines_deleted: 0, work_status: 'empty' }),
        access(2, { work_status: 'unknown', work_source: 'unknown' }),
        access(3, { work_status: 'binary' }),
        access(4, { work_status: 'unsupported_encoding' }),
      ],
      [node],
    )
    expect(geometry.segments).toHaveLength(0)
    expect(geometry.markers.map((marker) => marker.status)).toEqual([
      'empty',
      'unknown',
      'binary',
      'unsupported_encoding',
    ])
  })

  it('handles a degenerate node position deterministically', () => {
    const origin = { ...node, position: [0, 0, 0] as Vec3 }
    const segment = buildWorkGeometry(
      [access(1, { lines_added: 4, lines_deleted: 0, work_status: 'known' })],
      [origin],
    ).segments[0]!
    expect(segment.outward.every(Number.isFinite)).toBe(true)
    expect(segment.end.every(Number.isFinite)).toBe(true)
  })
})
