import { describe, expect, it } from 'vitest'
import { buildExtrusions } from '../src/activity/extrusions'
import type { GraphNode } from '../src/graph/types'

const nodes: GraphNode[] = [
  { id: 'root', path: '.', kind: 'root', position: [0, 0, 0] },
  {
    id: 'a',
    path: 'a.ts',
    kind: 'source',
    position: [7, 0, 0],
    activity: { total_time_ms: 60_000, lines_added: 10, lines_deleted: 2 },
  },
  {
    id: 'b',
    path: 'b.ts',
    kind: 'source',
    position: [0, 7, 0],
    activity: { total_time_ms: 15_000, lines_added: 2, lines_deleted: 0 },
  },
]

describe('activity extrusions', () => {
  it('extends radially outward with bounded relative lengths', () => {
    const extrusions = buildExtrusions(nodes, 'time')
    expect(extrusions).toHaveLength(2)
    expect(extrusions[0]?.end[0]).toBeCloseTo(11)
    expect(extrusions[0]?.end[1]).toBe(0)
    expect(extrusions[1]?.length).toBeLessThan(extrusions[0]?.length ?? 0)
  })

  it('uses reported line work independently from time', () => {
    const work = buildExtrusions(nodes, 'work')
    expect(work[0]?.value).toBe(12)
    expect(work[1]?.value).toBe(2)
  })
})
