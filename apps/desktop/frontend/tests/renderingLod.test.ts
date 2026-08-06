import { describe, expect, it } from 'vitest'
import type { AccessPoint } from '../src/activity/accessPoints'
import type { TrailAccess } from '../src/graph/types'
import {
  nodeGeometryDetail,
  markerGeometrySegments,
  selectAccessPointsForRender,
  selectActivityAccesses,
  shouldShowHoverLabel,
} from '../src/rendering/lod'

function access(sequence: number): TrailAccess {
  return {
    sequence,
    node_id: `node-${sequence % 20}`,
    path: `src/file-${sequence % 20}.ts`,
    started_at: new Date(Date.UTC(2026, 6, 25, 0, 0, sequence)).toISOString(),
    ended_at:
      sequence === 598
        ? undefined
        : new Date(Date.UTC(2026, 6, 25, 0, 0, sequence + 1)).toISOString(),
    duration_ms: sequence === 17 ? 1_000_000 : sequence * 10,
    operations: ['patch'],
    source: 'fixture',
    confidence: 'exact',
    lines_added: sequence === 23 ? 50_000 : sequence,
    lines_deleted: sequence === 29 ? 25_000 : sequence % 7,
    work_status: 'known',
  }
}

function point(sequence: number): AccessPoint {
  const latest = access(sequence)
  return {
    nodeId: latest.node_id!,
    position: [sequence, 0, 0],
    sequences: [sequence],
    accessCount: 1,
    aggregate: sequence % 5 === 0,
    active: sequence === 98,
    latest,
  }
}

describe('rendering LOD', () => {
  it('reduces node detail and hover-label range at deterministic thresholds', () => {
    expect(nodeGeometryDetail(1_000)).toBe(2)
    expect(nodeGeometryDetail(1_001)).toBe(1)
    expect(nodeGeometryDetail(5_001)).toBe(0)
    expect(markerGeometrySegments(48)).toBe(10)
    expect(markerGeometrySegments(49)).toBe(8)
    expect(markerGeometrySegments(129)).toBe(6)
    expect(shouldShowHoverLabel(1_000, 36)).toBe(true)
    expect(shouldShowHoverLabel(1_001, 23)).toBe(false)
    expect(shouldShowHoverLabel(5_001, Number.NaN)).toBe(false)
  })

  it('bounds dense activity while retaining active, selected, and scale-defining evidence', () => {
    const trail = Array.from({ length: 600 }, (_, index) => access(index + 1))
    const result = selectActivityAccesses(trail, 41, 64)

    expect(result).toMatchObject({
      totalCount: 600,
      omittedCount: 536,
      active: true,
    })
    expect(result.items).toHaveLength(64)
    expect(result.items.map((item) => item.sequence)).toEqual(
      [...result.items]
        .sort((left, right) => left.sequence - right.sequence)
        .map((item) => item.sequence),
    )
    expect(result.items.map((item) => item.sequence)).toEqual(
      expect.arrayContaining([17, 23, 29, 41, 598, 600]),
    )
    expect(selectActivityAccesses(trail, 41, 64)).toEqual(result)
  })

  it('bounds point instances while retaining inspected, active, aggregate, and recent points', () => {
    const points = Array.from({ length: 100 }, (_, index) => point(index + 1))
    const result = selectAccessPointsForRender(points, 7, 16)
    const sequences = result.items.map((item) => item.latest.sequence)

    expect(result).toMatchObject({
      totalCount: 100,
      omittedCount: 84,
      active: true,
    })
    expect(sequences).toEqual(expect.arrayContaining([7, 98, 99, 100]))
    expect(result.items.some((item) => item.aggregate)).toBe(true)
    expect(selectAccessPointsForRender(points, 7, 16)).toEqual(result)
  })
})
