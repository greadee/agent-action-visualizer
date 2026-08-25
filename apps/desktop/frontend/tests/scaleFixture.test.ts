import { describe, expect, it } from 'vitest'
import { buildAccessPoints } from '../src/activity/accessPoints'
import {
  MAX_ACCESS_POINT_INSTANCES,
  MAX_ACTIVITY_ACCESSES,
  selectAccessPointsForRender,
  selectActivityAccesses,
} from '../src/rendering/lod'
import {
  MAX_SCALE_HISTORY,
  SCALE_NODE_COUNTS,
  createScaleFixture,
  scaleFixtureFromSearch,
} from '../src/rendering/scaleFixture'
import { buildStructurePositions } from '../src/rendering/structurePositions'

describe('scale fixtures', () => {
  it.each(SCALE_NODE_COUNTS)(
    'builds a deterministic connected %,i-node graph',
    (nodeCount) => {
      const first = createScaleFixture({ nodeCount, accessCount: 12 })
      const second = createScaleFixture({ nodeCount, accessCount: 12 })

      expect(first.graph.nodes).toHaveLength(nodeCount)
      expect(first.graph.edges).toHaveLength(nodeCount - 1)
      expect(new Set(first.graph.nodes.map((node) => node.id)).size).toBe(
        nodeCount,
      )
      expect(first).toEqual(second)
      expect(
        first.graph.nodes.every((node) =>
          node.position.every((coordinate) => Number.isFinite(coordinate)),
        ),
      ).toBe(true)
    },
  )

  it('bounds long-session detail without changing exact history', () => {
    const fixture = createScaleFixture({
      nodeCount: 5_000,
      accessCount: MAX_SCALE_HISTORY,
    })
    const activity = selectActivityAccesses(fixture.focus.trail)
    const points = selectAccessPointsForRender(
      buildAccessPoints(fixture.focus.trail, fixture.graph.nodes),
    )

    expect(fixture.focus.trail).toHaveLength(MAX_SCALE_HISTORY)
    expect(activity.items).toHaveLength(MAX_ACTIVITY_ACCESSES)
    expect(activity.totalCount).toBe(MAX_SCALE_HISTORY)
    expect(points.items).toHaveLength(MAX_ACCESS_POINT_INSTANCES)
    expect(points.totalCount).toBeGreaterThan(MAX_ACCESS_POINT_INSTANCES)
    expect(activity.items.at(-1)?.ended_at).toBeUndefined()
  })

  it('accepts only campaign node counts and clamps optional history', () => {
    expect(
      scaleFixtureFromSearch('?scale=5000&history=75000', true),
    ).toMatchObject({
      nodeCount: 5_000,
      accessCount: 75_000,
      diagnostics: true,
    })
    expect(
      scaleFixtureFromSearch('?scale=20000&history=999999&diagnostics=0', true),
    ).toMatchObject({
      nodeCount: 20_000,
      accessCount: MAX_SCALE_HISTORY,
      diagnostics: false,
    })
    expect(scaleFixtureFromSearch('?scale=4999', true)).toBeUndefined()
    expect(scaleFixtureFromSearch('?scale=5000', false)).toBeUndefined()
  })

  it('fills one fixed-size structure buffer without intermediate arrays', () => {
    const fixture = createScaleFixture({ nodeCount: 100, accessCount: 0 })
    const first = buildStructurePositions(
      fixture.graph.nodes,
      fixture.graph.edges,
    )
    const second = buildStructurePositions(
      fixture.graph.nodes,
      fixture.graph.edges,
    )

    expect(first).toHaveLength(fixture.graph.edges.length * 6)
    expect(first).toEqual(second)
  })
})
