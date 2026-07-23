import { describe, expect, it } from 'vitest'
import type { GraphNode, TrailAccess } from '../src/graph/types'
import {
  buildTrailSegments,
  formatTrailTimestamp,
  selectTrailAccesses,
} from '../src/scene/trail'

const nodes: GraphNode[] = ['a', 'b', 'c', 'd', 'e'].map((id, index) => ({
  id,
  path: `${id}.go`,
  kind: 'source',
  position: [index, index * 2, 0],
}))

const trail: TrailAccess[] = nodes.map((node, index) => ({
  sequence: index + 1,
  node_id: node.id,
  path: node.path,
  started_at: `2026-07-22T00:00:0${index}Z`,
  duration_ms: 1000,
  operations: [index % 2 === 0 ? 'read' : 'patch'],
  source: 'synthetic',
  confidence: 'exact',
}))

describe('session trail', () => {
  it('limits by recent accesses while preserving direction and metadata', () => {
    const segments = buildTrailSegments(trail, nodes, {
      recentAccesses: 3,
      completeSession: false,
      hideConsecutiveRepeats: true,
    })
    expect(
      segments.map(({ sourceId, targetId }) => [sourceId, targetId]),
    ).toEqual([
      ['c', 'd'],
      ['d', 'e'],
    ])
    expect(segments[1]).toMatchObject({
      toSequence: 5,
      operation: 'read',
      confidence: 'exact',
      recency: 1,
    })
    expect(segments[1]!.arrow[0]).toBeCloseTo(3.72)
    expect(segments[1]!.arrow[1]).toBeCloseTo(7.44)
  })

  it('shows the complete session and can collapse consecutive repeats', () => {
    const repeated = [
      trail[0]!,
      { ...trail[0]!, sequence: 2 },
      { ...trail[1]!, sequence: 3 },
    ]
    expect(
      selectTrailAccesses(repeated, {
        recentAccesses: 2,
        completeSession: true,
        hideConsecutiveRepeats: true,
      }).map((access) => access.node_id),
    ).toEqual(['a', 'b'])
    expect(
      selectTrailAccesses(repeated, {
        recentAccesses: 2,
        completeSession: true,
        hideConsecutiveRepeats: false,
      }),
    ).toHaveLength(3)
  })

  it('formats valid timestamps deterministically and preserves invalid input', () => {
    expect(formatTrailTimestamp('2026-07-22T01:02:03Z')).toBe(
      '2026-07-22 01:02:03Z',
    )
    expect(formatTrailTimestamp('unknown')).toBe('unknown')
  })
})
