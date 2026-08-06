import { accessAnchor } from './accessPoints'
import type { GraphNode, TrailAccess, Vec3 } from '../graph/types'
import { addScaled, normalize, subtract } from '../rendering/vectorMath'

export type DurationScale = 'linear' | 'log'

export interface TimeExtrusion {
  nodeId: string
  sequence: number
  access: TrailAccess
  start: Vec3
  end: Vec3
  outward: Vec3
  durationMs: number
  length: number
  clamped: boolean
  active: boolean
}

export const ACTIVE_INTERVAL_CAP_MS = 2 * 60 * 1000
export const MAX_DURATION_SCALE_MS = 60 * 1000
export const MIN_DURATION_EXTRUSION = 0.18
export const MAX_DURATION_EXTRUSION = 2.8

function validDuration(value: number): number {
  return Number.isFinite(value) ? Math.max(0, value) : 0
}

export function normalizedIntervalDuration(
  access: TrailAccess,
  nowMs: number,
): number {
  const reported = validDuration(access.duration_ms)
  if (access.ended_at) return reported
  const startedAt = Date.parse(access.started_at)
  if (!Number.isFinite(startedAt) || !Number.isFinite(nowMs)) return reported
  const elapsed = Math.max(0, nowMs - startedAt)
  return Math.max(reported, Math.min(elapsed, ACTIVE_INTERVAL_CAP_MS))
}

export function durationScaleRatio(
  durationMs: number,
  maximumMs: number,
  scale: DurationScale,
): number {
  if (durationMs <= 0 || maximumMs <= 0) return 0
  const unclamped =
    scale === 'linear'
      ? durationMs / maximumMs
      : Math.log1p(durationMs) / Math.log1p(maximumMs)
  return Math.min(1, Math.max(0, unclamped))
}

export function durationExtrusionLength(
  durationMs: number,
  maximumMs: number,
  scale: DurationScale,
): number {
  if (durationMs <= 0) return 0
  return (
    MIN_DURATION_EXTRUSION +
    (MAX_DURATION_EXTRUSION - MIN_DURATION_EXTRUSION) *
      durationScaleRatio(durationMs, maximumMs, scale)
  )
}

function outwardFromAnchor(anchor: Vec3, node: GraphNode): Vec3 {
  return normalize(subtract(anchor, node.position))
}

export function buildTimeExtrusions(
  trail: readonly TrailAccess[],
  nodes: readonly GraphNode[],
  nowMs: number,
  scale: DurationScale = 'log',
  visualCapMs: number = MAX_DURATION_SCALE_MS,
): TimeExtrusion[] {
  const cap = Number.isFinite(visualCapMs)
    ? Math.max(1, visualCapMs)
    : MAX_DURATION_SCALE_MS
  const nodesByID = new Map(nodes.map((node) => [node.id, node]))
  const candidates = [...trail]
    .sort((left, right) => left.sequence - right.sequence)
    .flatMap((access) => {
      const node = access.node_id ? nodesByID.get(access.node_id) : undefined
      if (!node || node.kind === 'root' || node.kind === 'directory') return []
      const durationMs = normalizedIntervalDuration(access, nowMs)
      if (durationMs <= 0) return []
      return [{ access, node, durationMs }]
    })
  const maximumMs = Math.min(
    cap,
    Math.max(0, ...candidates.map(({ durationMs }) => durationMs)),
  )

  return candidates.map(({ access, node, durationMs }) => {
    const start = accessAnchor(node.position, Math.max(0, access.sequence - 1))
    const outward = outwardFromAnchor(start, node)
    const unclampedLength = durationExtrusionLength(
      durationMs,
      maximumMs,
      scale,
    )
    const length = Math.min(
      MAX_DURATION_EXTRUSION,
      Math.max(MIN_DURATION_EXTRUSION, unclampedLength),
    )
    const end = addScaled(start, outward, length)
    return {
      nodeId: node.id,
      sequence: access.sequence,
      access,
      start,
      end,
      outward,
      durationMs,
      length,
      clamped: durationMs > cap,
      active: !access.ended_at,
    }
  })
}
