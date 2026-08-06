import { bench, describe } from 'vitest'
import { buildAccessPoints } from '../src/activity/accessPoints'
import { buildSessionAnalytics } from '../src/analytics/sessionAnalytics'
import {
  selectAccessPointsForRender,
  selectActivityAccesses,
} from '../src/rendering/lod'
import {
  SCALE_NODE_COUNTS,
  createScaleFixture,
} from '../src/rendering/scaleFixture'
import { buildStructurePositions } from '../src/rendering/structurePositions'

describe('whole-app scale campaign', () => {
  for (const nodeCount of SCALE_NODE_COUNTS) {
    bench(`build ${nodeCount.toLocaleString()}-node render input`, () => {
      const fixture = createScaleFixture({ nodeCount, accessCount: 0 })
      buildStructurePositions(fixture.graph.nodes, fixture.graph.edges)
    })
  }

  const longSession = createScaleFixture({
    nodeCount: 5_000,
    accessCount: 100_000,
  })
  bench('select 100,000-access long-session detail', () => {
    selectActivityAccesses(longSession.focus.trail)
    selectAccessPointsForRender(
      buildAccessPoints(longSession.focus.trail, longSession.graph.nodes),
    )
  })
  bench('calculate 100,000-access analytics', () => {
    buildSessionAnalytics(longSession.graph, longSession.focus)
  })
})
