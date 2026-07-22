import { nodeColors } from '../graph/palette'
import type { GraphNode } from '../graph/types'

const kinds: GraphNode['kind'][] = [
  'root',
  'directory',
  'source',
  'test',
  'config',
  'documentation',
  'asset',
  'tombstone',
]

export function GraphLegend() {
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
    </div>
  )
}
