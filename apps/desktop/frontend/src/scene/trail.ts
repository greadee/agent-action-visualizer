import type { GraphNode, TrailAccess, Vec3 } from '../graph/types'

export interface TrailOptions {
  recentAccesses: number
  completeSession: boolean
  hideConsecutiveRepeats: boolean
}

export interface TrailSegment {
  fromSequence: number
  toSequence: number
  sourceId: string
  targetId: string
  start: Vec3
  end: Vec3
  midpoint: Vec3
  arrow: Vec3
  operation: string
  timestamp: string
  source: string
  confidence: string
  recency: number
}

function accessIdentity(access: TrailAccess): string {
  return access.node_id || access.path
}

export function selectTrailAccesses(
  trail: readonly TrailAccess[],
  options: TrailOptions,
): TrailAccess[] {
  const ordered = [...trail].sort((a, b) => a.sequence - b.sequence)
  const collapsed = options.hideConsecutiveRepeats
    ? ordered.filter(
        (access, index) =>
          index === 0 ||
          accessIdentity(access) !== accessIdentity(ordered[index - 1]!),
      )
    : ordered
  if (options.completeSession) return collapsed
  const requested = Number.isFinite(options.recentAccesses)
    ? Math.floor(options.recentAccesses)
    : 2
  const limit = Math.min(200, Math.max(2, requested))
  return collapsed.slice(-limit)
}

export function buildTrailSegments(
  trail: readonly TrailAccess[],
  nodes: readonly GraphNode[],
  options: TrailOptions,
): TrailSegment[] {
  const visible = selectTrailAccesses(trail, options)
  const positions = new Map(nodes.map((node) => [node.id, node.position]))
  const candidates = visible.slice(1).map((target, index) => ({
    source: visible[index]!,
    target,
  }))
  const segments = candidates.flatMap(({ source, target }) => {
    const sourceId = source.node_id ?? ''
    const targetId = target.node_id ?? ''
    const start = positions.get(sourceId)
    const end = positions.get(targetId)
    if (!start || !end || sourceId === targetId) return []
    const point = (ratio: number): Vec3 => [
      start[0] + (end[0] - start[0]) * ratio,
      start[1] + (end[1] - start[1]) * ratio,
      start[2] + (end[2] - start[2]) * ratio,
    ]
    return [
      {
        fromSequence: source.sequence,
        toSequence: target.sequence,
        sourceId,
        targetId,
        start,
        end,
        midpoint: point(0.5),
        arrow: point(0.72),
        operation: target.operations.at(-1) ?? 'access',
        timestamp: target.started_at,
        source: target.source,
        confidence: target.confidence,
        recency: 0,
      },
    ]
  })
  return segments.map((segment, index) => ({
    ...segment,
    recency: (index + 1) / segments.length,
  }))
}

export function formatTrailTimestamp(value: string): string {
  const timestamp = new Date(value)
  return Number.isNaN(timestamp.valueOf())
    ? value
    : timestamp.toISOString().replace('T', ' ').replace('.000Z', 'Z')
}
