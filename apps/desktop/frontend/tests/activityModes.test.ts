import { describe, expect, it } from 'vitest'
import { buildAccessPoints } from '../src/activity/accessPoints'
import { buildTimeExtrusions } from '../src/activity/timeExtrusions'
import { buildWorkGeometry } from '../src/activity/workExtrusions'
import type { GraphNode, TrailAccess } from '../src/graph/types'

const node: GraphNode = {
  id: 'file-a',
  path: 'src/a.ts',
  kind: 'source',
  position: [4, 3, 5],
}

const trail: TrailAccess[] = [
  {
    sequence: 1,
    node_id: node.id,
    path: node.path,
    started_at: '2026-07-25T12:00:00Z',
    ended_at: '2026-07-25T12:00:10Z',
    duration_ms: 10_000,
    operations: ['patch'],
    source: 'synthetic',
    confidence: 'exact',
    lines_added: 24,
    lines_deleted: 8,
    work_status: 'known',
  },
]

describe('activity mode geometry', () => {
  it('keeps access anchors stable when changing modes and replaying', () => {
    const time = buildTimeExtrusions(
      trail,
      [node],
      Date.parse('2026-07-25T12:01:00Z'),
    )
    const work = buildWorkGeometry(trail, [node])
    const replay = buildWorkGeometry(trail, [node])

    expect(time[0]?.start).toEqual(work.segments[0]?.start)
    expect(work).toEqual(replay)
  })

  it('keeps each marker and extrusion on the same file-local anchor', () => {
    const otherNode: GraphNode = {
      id: 'file-b',
      path: 'src/b.ts',
      kind: 'source',
      position: [-4, 2, 3],
    }
    const interleaved: TrailAccess[] = [
      {
        ...trail[0]!,
        sequence: 1,
        node_id: otherNode.id,
        path: otherNode.path,
      },
      { ...trail[0]!, sequence: 2 },
      { ...trail[0]!, sequence: 3, duration_ms: 20_000 },
    ]
    const nodes = [node, otherNode]
    const points = buildAccessPoints(interleaved, nodes)
    const time = buildTimeExtrusions(
      interleaved,
      nodes,
      Date.parse('2026-07-25T12:01:00Z'),
    )
    const work = buildWorkGeometry(interleaved, nodes)

    for (const sequence of [2, 3]) {
      const point = points.find((item) => item.sequences.includes(sequence))
      const timeStart = time.find((item) => item.sequence === sequence)?.start
      const workStarts = work.segments
        .filter((item) => item.sequence === sequence)
        .map((item) => item.start)
      expect(point?.position).toEqual(timeStart)
      expect(workStarts.length).toBeGreaterThan(0)
      expect(workStarts.every((start) => start === workStarts[0])).toBe(true)
      expect(workStarts[0]).toEqual(point?.position)
    }
    expect(
      points.find((item) => item.sequences.includes(2))?.position,
    ).not.toEqual(points.find((item) => item.sequences.includes(3))?.position)
  })
})
