import { VisuallyHidden } from '@prool-ui/react'
import { nodeColors } from '../graph/palette'
import type { GraphNode } from '../graph/types'
import type { ActivityMode } from '../activity/extrusions'
import {
  activityVisualCap,
  formatVisualCap,
  type ActivityDisplaySettings,
} from '../activity/displaySettings'

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

export function GraphLegend({
  activityMode,
  activitySettings,
}: {
  activityMode: ActivityMode
  activitySettings: ActivityDisplaySettings
}) {
  const visualCap = formatVisualCap(
    activityMode,
    activityVisualCap(activityMode, activitySettings),
  )
  return (
    <div className="legend" aria-labelledby="graph-legend-heading">
      <VisuallyHidden id="graph-legend-heading">Graph legend</VisuallyHidden>
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
        <i className="legend__trail" /> session travel → newest
      </span>
      <span className="legend__wide">
        <i className={`legend__activity legend__activity--${activityMode}`} />
        {activityMode === 'time'
          ? 'outward line = access duration'
          : 'outward = additions · inward = deletions'}
      </span>
      <span className="legend__wide">
        {activitySettings.scale === 'log' ? 'Logarithmic' : 'Linear'} visual
        scale, capped at {visualCap}; exact values remain inspectable
      </span>
      <span>
        <i className="legend__access-point" /> access point
      </span>
      <span>
        <i className="legend__access-aggregate" /> aggregated accesses
      </span>
      <span className="legend__wide">Proximity = latest Git involvement</span>
      <span>
        <i style={{ background: '#35ffd2' }} /> current
      </span>
      <span>
        <i style={{ background: '#ffb45f' }} /> previous
      </span>
      <span className="legend__wide">
        <i style={{ background: '#d59aff' }} /> secondary-active
      </span>
      <span className="legend__wide">
        <i style={{ background: '#4d9aa0' }} /> older trail access
      </span>
    </div>
  )
}
