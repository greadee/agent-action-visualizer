import {
  accessAnchor,
  accessAnchorOrdinal,
  accessAnchorOrdinals,
} from './accessPoints'
import type { DurationScale } from './timeExtrusions'
import type { GraphNode, TrailAccess, Vec3 } from '../graph/types'
import { addScaled, normalize, subtract } from '../rendering/vectorMath'

export type WorkDirection = 'addition' | 'deletion'

export interface WorkSegment {
  nodeId: string
  sequence: number
  access: TrailAccess
  direction: WorkDirection
  start: Vec3
  end: Vec3
  outward: Vec3
  value: number
  length: number
  clamped: boolean
}

export interface WorkMarker {
  nodeId: string
  sequence: number
  access: TrailAccess
  position: Vec3
  status: NonNullable<TrailAccess['work_status']>
}

export interface WorkGeometry {
  segments: WorkSegment[]
  markers: WorkMarker[]
}

export const MIN_WORK_EXTRUSION = 0.18
export const MAX_WORK_EXTRUSION = 2.2
export const MAX_WORK_SCALE_LINES = 1_000

function workScaleRatio(
  value: number,
  maximum: number,
  scale: DurationScale,
): number {
  if (value <= 0 || maximum <= 0) return 0
  const ratio =
    scale === 'linear'
      ? value / maximum
      : Math.log1p(value) / Math.log1p(maximum)
  return Math.min(1, Math.max(0, ratio))
}

export function workExtrusionLength(
  value: number,
  maximum: number,
  scale: DurationScale = 'log',
): number {
  if (value <= 0) return 0
  return (
    MIN_WORK_EXTRUSION +
    (MAX_WORK_EXTRUSION - MIN_WORK_EXTRUSION) *
      workScaleRatio(value, maximum, scale)
  )
}

function outwardFromAnchor(anchor: Vec3, node: GraphNode): Vec3 {
  return normalize(subtract(anchor, node.position))
}

function workStatus(
  access: TrailAccess,
): NonNullable<TrailAccess['work_status']> {
  if (access.work_status) return access.work_status
  if (access.lines_added !== undefined || access.lines_deleted !== undefined)
    return (access.lines_added ?? 0) === 0 && (access.lines_deleted ?? 0) === 0
      ? 'empty'
      : 'known'
  return 'unknown'
}

export function buildWorkGeometry(
  trail: readonly TrailAccess[],
  nodes: readonly GraphNode[],
  scale: DurationScale = 'log',
  visualCapLines: number = MAX_WORK_SCALE_LINES,
): WorkGeometry {
  const cap = Number.isFinite(visualCapLines)
    ? Math.max(1, visualCapLines)
    : MAX_WORK_SCALE_LINES
  const nodesByID = new Map(nodes.map((node) => [node.id, node]))
  const ordinals = accessAnchorOrdinals(trail)
  const accesses = [...trail]
    .sort((left, right) => left.sequence - right.sequence)
    .flatMap((access) => {
      const node = access.node_id ? nodesByID.get(access.node_id) : undefined
      return node && node.kind !== 'root' && node.kind !== 'directory'
        ? [{ access, node }]
        : []
    })
  const maximum = Math.min(
    cap,
    Math.max(
      0,
      ...accesses.flatMap(({ access }) => [
        Math.max(0, access.lines_added ?? 0),
        Math.max(0, access.lines_deleted ?? 0),
      ]),
    ),
  )
  const segments: WorkSegment[] = []
  const markers: WorkMarker[] = []

  for (const { access, node } of accesses) {
    const start = accessAnchor(
      node.position,
      accessAnchorOrdinal(ordinals, access),
    )
    const outward = outwardFromAnchor(start, node)
    const status = workStatus(access)
    if (status !== 'known') {
      markers.push({
        nodeId: node.id,
        sequence: access.sequence,
        access,
        position: start,
        status,
      })
      continue
    }
    const values: Array<[WorkDirection, number]> = [
      ['addition', Math.max(0, access.lines_added ?? 0)],
      ['deletion', Math.max(0, access.lines_deleted ?? 0)],
    ]
    for (const [direction, value] of values) {
      if (value === 0) continue
      const length = workExtrusionLength(value, maximum, scale)
      const sign = direction === 'addition' ? 1 : -1
      const end = addScaled(start, outward, length * sign)
      segments.push({
        nodeId: node.id,
        sequence: access.sequence,
        access,
        direction,
        start,
        end,
        outward,
        value,
        length,
        clamped: value > cap,
      })
    }
  }
  return { segments, markers }
}
