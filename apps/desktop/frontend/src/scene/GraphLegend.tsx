import { nodeColors } from '../graph/palette'
import type { GraphNode } from '../graph/types'
import type { ActivityMode } from '../activity/extrusions'

const kinds: GraphNode['kind'][] = [
  'root',
  'directory',
  'source',
  'test',
  'config',
  'documentation',
  'asset',
  'generated',
  'tombstone',
]

export function GraphLegend({ activityMode }: { activityMode: ActivityMode }) {
  return (
    <div className="legend" aria-label="Graph legend">
      <p className="panel__label">LEGEND</p>
      {kinds.map((kind) => (
        <span key={kind}>
          <i style={{ background: nodeColors[kind] }} /> {kind}
        </span>
      ))}
      <span>
        <i className="legend__edge" /> structure edge
      </span>
      <span className="legend__wide">
        <i className={`legend__activity legend__activity--${activityMode}`} />
        outward line = reported {activityMode}
      </span>
      <span className="legend__wide">Proximity = latest Git involvement</span>
    </div>
  )
}
