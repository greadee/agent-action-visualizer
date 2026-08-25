import { bench, describe } from 'vitest'
import type { AccessPoint } from '../src/activity/accessPoints'
import type { TrailAccess } from '../src/graph/types'
import {
  selectAccessPointsForRender,
  selectActivityAccesses,
} from '../src/rendering/lod'

const trails = new Map(
  [1_000, 10_000].map((count) => [
    count,
    Array.from({ length: count }, (_, index) => fixtureAccess(index + 1)),
  ]),
)
const points = Array.from({ length: 10_000 }, (_, index) =>
  fixturePoint(index + 1),
)

describe('render-model LOD', () => {
  bench('select 1,000 activity accesses', () => {
    selectActivityAccesses(trails.get(1_000)!, 500)
  })

  bench('select 10,000 activity accesses', () => {
    selectActivityAccesses(trails.get(10_000)!, 5_000)
  })

  bench('select 10,000 access points', () => {
    selectAccessPointsForRender(points, 5_000)
  })
})

function fixtureAccess(sequence: number): TrailAccess {
  return {
    sequence,
    node_id: `node-${sequence % 2_000}`,
    path: `src/file-${sequence % 2_000}.ts`,
    started_at: '2026-07-25T00:00:00Z',
    ended_at: sequence === 10_000 ? undefined : '2026-07-25T00:00:01Z',
    duration_ms: sequence * 10,
    operations: ['patch'],
    source: 'benchmark',
    confidence: 'exact',
    lines_added: sequence % 500,
    lines_deleted: sequence % 100,
    work_status: 'known',
  }
}

function fixturePoint(sequence: number): AccessPoint {
  const latest = fixtureAccess(sequence)
  return {
    nodeId: latest.node_id!,
    position: [sequence % 100, sequence % 50, sequence % 25],
    sequences: [sequence],
    accessCount: 1,
    aggregate: sequence % 25 === 0,
    active: !latest.ended_at,
    latest,
  }
}
