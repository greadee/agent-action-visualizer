import type { GraphNode, TrailAccess, Vec3 } from '../graph/types'

export const MAX_INDIVIDUAL_ACCESS_POINTS = 24
const RECENT_INDIVIDUAL_ACCESS_POINTS = 16
const MAX_AGGREGATE_POINTS = 8
const NODE_SURFACE_OFFSET = 0.29
const GOLDEN_ANGLE = Math.PI * (3 - Math.sqrt(5))
const INVERSE_GOLDEN_RATIO = (Math.sqrt(5) - 1) / 2

export interface AccessPoint {
  nodeId: string
  position: Vec3
  sequences: number[]
  accessCount: number
  aggregate: boolean
  active: boolean
  latest: TrailAccess
}

function dot(a: Vec3, b: Vec3): number {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

function cross(a: Vec3, b: Vec3): Vec3 {
  return [
    a[1] * b[2] - a[2] * b[1],
    a[2] * b[0] - a[0] * b[2],
    a[0] * b[1] - a[1] * b[0],
  ]
}

function normalize(vector: Vec3): Vec3 {
  const length = Math.hypot(...vector)
  return length === 0
    ? [0, 0, 1]
    : [vector[0] / length, vector[1] / length, vector[2] / length]
}

function outwardBasis(position: Vec3): [Vec3, Vec3, Vec3] {
  const outward = normalize(position)
  const reference: Vec3 = Math.abs(outward[1]) > 0.9 ? [1, 0, 0] : [0, 1, 0]
  const tangentA = normalize(cross(outward, reference))
  return [outward, tangentA, normalize(cross(outward, tangentA))]
}

export function accessAnchor(position: Vec3, ordinal: number): Vec3 {
  const [outward, tangentA, tangentB] = outwardBasis(position)
  const radius =
    ordinal === 0 ? 0 : 0.82 * Math.sqrt((ordinal * INVERSE_GOLDEN_RATIO) % 1)
  const angle = ordinal * GOLDEN_ANGLE
  const forward = Math.sqrt(1 - radius * radius)
  const normal: Vec3 = normalize([
    outward[0] * forward +
      tangentA[0] * Math.cos(angle) * radius +
      tangentB[0] * Math.sin(angle) * radius,
    outward[1] * forward +
      tangentA[1] * Math.cos(angle) * radius +
      tangentB[1] * Math.sin(angle) * radius,
    outward[2] * forward +
      tangentA[2] * Math.cos(angle) * radius +
      tangentB[2] * Math.sin(angle) * radius,
  ])
  return [
    position[0] + normal[0] * NODE_SURFACE_OFFSET,
    position[1] + normal[1] * NODE_SURFACE_OFFSET,
    position[2] + normal[2] * NODE_SURFACE_OFFSET,
  ]
}

function pointFromAccesses(
  node: GraphNode,
  accesses: readonly TrailAccess[],
  ordinal: number,
  aggregate: boolean,
): AccessPoint {
  const latest = accesses[accesses.length - 1]!
  return {
    nodeId: node.id,
    position: accessAnchor(node.position, ordinal),
    sequences: accesses.map((access) => access.sequence),
    accessCount: accesses.length,
    aggregate,
    active: !latest.ended_at,
    latest,
  }
}

export function buildAccessPoints(
  trail: readonly TrailAccess[],
  nodes: readonly GraphNode[],
): AccessPoint[] {
  const nodeByID = new Map(nodes.map((node) => [node.id, node]))
  const accessesByNode = new Map<string, TrailAccess[]>()
  for (const access of [...trail].sort((a, b) => a.sequence - b.sequence)) {
    if (!access.node_id || !nodeByID.has(access.node_id)) continue
    const accesses = accessesByNode.get(access.node_id) ?? []
    accesses.push(access)
    accessesByNode.set(access.node_id, accesses)
  }

  const points: AccessPoint[] = []
  for (const [nodeID, accesses] of accessesByNode) {
    const node = nodeByID.get(nodeID)!
    if (accesses.length <= MAX_INDIVIDUAL_ACCESS_POINTS) {
      accesses.forEach((access, ordinal) =>
        points.push(pointFromAccesses(node, [access], ordinal, false)),
      )
      continue
    }

    const recent = accesses.slice(-RECENT_INDIVIDUAL_ACCESS_POINTS)
    const older = accesses.slice(0, -RECENT_INDIVIDUAL_ACCESS_POINTS)
    const bucketSize = Math.ceil(older.length / MAX_AGGREGATE_POINTS)
    for (let start = 0; start < older.length; start += bucketSize) {
      points.push(
        pointFromAccesses(
          node,
          older.slice(start, start + bucketSize),
          start / bucketSize,
          true,
        ),
      )
    }
    recent.forEach((access, index) =>
      points.push(
        pointFromAccesses(node, [access], older.length + index, false),
      ),
    )
  }
  return points
}

export function anchorDistanceFromNode(
  point: AccessPoint,
  node: GraphNode,
): number {
  return Math.hypot(
    point.position[0] - node.position[0],
    point.position[1] - node.position[1],
    point.position[2] - node.position[2],
  )
}

export function isOutwardAnchor(point: AccessPoint, node: GraphNode): boolean {
  const offset: Vec3 = [
    point.position[0] - node.position[0],
    point.position[1] - node.position[1],
    point.position[2] - node.position[2],
  ]
  return dot(offset, node.position) > 0
}
