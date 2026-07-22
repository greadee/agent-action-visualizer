import { Vector3 } from 'three'
import type { GraphNode, NodeActivity, Vec3 } from '../graph/types'

export type ActivityMode = 'time' | 'work'

export interface Extrusion {
  nodeId: string
  start: Vec3
  end: Vec3
  value: number
  length: number
}

export function activityValue(
  activity: NodeActivity | undefined,
  mode: ActivityMode,
): number {
  if (!activity) return 0
  return mode === 'time'
    ? Math.max(0, activity.total_time_ms ?? 0)
    : Math.max(0, activity.lines_added ?? 0) +
        Math.max(0, activity.lines_deleted ?? 0)
}

export function buildExtrusions(
  nodes: GraphNode[],
  mode: ActivityMode,
): Extrusion[] {
  const files = nodes.filter(
    (node) => node.kind !== 'root' && node.kind !== 'directory',
  )
  const maximum = Math.max(
    0,
    ...files.map((node) => activityValue(node.activity, mode)),
  )
  if (maximum === 0) return []
  return files.flatMap((node) => {
    const value = activityValue(node.activity, mode)
    if (value === 0) return []
    const length = 0.45 + (3.55 * Math.log1p(value)) / Math.log1p(maximum)
    const start = new Vector3(...node.position)
    const radial = start.clone().normalize()
    const end = start.clone().addScaledVector(radial, length)
    return [
      {
        nodeId: node.id,
        start: start.toArray() as unknown as Vec3,
        end: end.toArray() as unknown as Vec3,
        value,
        length,
      },
    ]
  })
}
