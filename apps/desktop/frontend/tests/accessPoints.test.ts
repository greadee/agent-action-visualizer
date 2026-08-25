import { describe, expect, it } from 'vitest'
import {
  accessAnchor,
  anchorDistanceFromNode,
  buildAccessPoints,
  isOutwardAnchor,
} from '../src/activity/accessPoints'
import type { GraphNode, TrailAccess } from '../src/graph/types'

const node: GraphNode = {
  id: 'node-a',
  path: 'src/a.ts',
  kind: 'source',
  position: [4, 3, 5],
}

function access(sequence: number): TrailAccess {
  return {
    sequence,
    node_id: node.id,
    path: node.path,
    started_at: `2026-07-22T00:00:${String(sequence).padStart(2, '0')}Z`,
    duration_ms: 1000,
    operations: ['read'],
    source: 'synthetic',
    confidence: 'exact',
  }
}

describe('access points', () => {
  it('uses stable outward-facing Fibonacci-disk anchors', () => {
    expect(accessAnchor(node.position, 4)).toEqual(
      accessAnchor(node.position, 4),
    )
    expect(accessAnchor(node.position, 1)).not.toEqual(
      accessAnchor(node.position, 2),
    )
    const point = buildAccessPoints([access(1)], [node])[0]!
    expect(anchorDistanceFromNode(point, node)).toBeCloseTo(0.29)
    expect(isOutwardAnchor(point, node)).toBe(true)
  })

  it('creates one point for every ordinary access interval', () => {
    const points = buildAccessPoints([access(3), access(1), access(2)], [node])
    expect(points.map((point) => point.sequences)).toEqual([[1], [2], [3]])
    expect(points.every((point) => !point.aggregate)).toBe(true)
  })

  it('aggregates dense history while retaining its full access count and recent points', () => {
    const trail = Array.from({ length: 40 }, (_, index) => access(index + 1))
    const points = buildAccessPoints(trail, [node])
    expect(points.filter((point) => point.aggregate)).toHaveLength(8)
    expect(points.filter((point) => !point.aggregate)).toHaveLength(16)
    expect(points.reduce((total, point) => total + point.accessCount, 0)).toBe(
      40,
    )
    expect(points.at(-1)?.sequences).toEqual([40])
  })
})
