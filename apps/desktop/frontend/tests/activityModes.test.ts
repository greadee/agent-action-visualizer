import { describe, expect, it } from 'vitest'
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
})
