import type { AccessPoint } from '../activity/accessPoints'
import type { TrailAccess } from '../graph/types'

export const MAX_ACTIVITY_ACCESSES = 256
export const MAX_ACCESS_POINT_INSTANCES = 2_048

export interface LodSelection<T> {
  items: T[]
  totalCount: number
  omittedCount: number
  active: boolean
}

export function nodeGeometryDetail(nodeCount: number): 0 | 1 | 2 {
  if (nodeCount > 5_000) return 0
  if (nodeCount > 1_000) return 1
  return 2
}

export function markerGeometrySegments(instanceCount: number): 6 | 8 | 10 {
  if (instanceCount > 128) return 6
  if (instanceCount > 48) return 8
  return 10
}

export function shouldShowHoverLabel(
  nodeCount: number,
  cameraDistance: number,
): boolean {
  if (!Number.isFinite(cameraDistance)) return false
  const maximumDistance = nodeCount > 5_000 ? 14 : nodeCount > 1_000 ? 22 : 36
  return cameraDistance <= maximumDistance
}

export function selectActivityAccesses(
  trail: readonly TrailAccess[],
  selectedSequence?: number,
  limit = MAX_ACTIVITY_ACCESSES,
): LodSelection<TrailAccess> {
  const ordered = [...trail].sort(
    (left, right) => left.sequence - right.sequence,
  )
  const boundedLimit = normalizeLimit(limit)
  if (ordered.length <= boundedLimit) return selection(ordered, ordered.length)

  const chosen = new Map<number, TrailAccess>()
  const add = (access: TrailAccess | undefined) => {
    if (access && chosen.size < boundedLimit)
      chosen.set(access.sequence, access)
  }

  add(ordered.find((access) => access.sequence === selectedSequence))
  ordered
    .filter((access) => !access.ended_at)
    .reverse()
    .forEach(add)
  add(maximumAccess(ordered, (access) => access.duration_ms))
  add(maximumAccess(ordered, (access) => access.lines_added ?? 0))
  add(maximumAccess(ordered, (access) => access.lines_deleted ?? 0))

  const recentTarget = Math.max(chosen.size, Math.floor(boundedLimit * 0.75))
  for (
    let index = ordered.length - 1;
    index >= 0 && chosen.size < recentTarget;
    index -= 1
  )
    add(ordered[index])

  const remaining = ordered.filter((access) => !chosen.has(access.sequence))
  const slots = boundedLimit - chosen.size
  evenlySpaced(remaining, slots).forEach(add)

  return selection(
    [...chosen.values()].sort((left, right) => left.sequence - right.sequence),
    ordered.length,
  )
}

export function selectAccessPointsForRender(
  points: readonly AccessPoint[],
  selectedSequence?: number,
  limit = MAX_ACCESS_POINT_INSTANCES,
): LodSelection<AccessPoint> {
  const ordered = [...points].sort(
    (left, right) => left.latest.sequence - right.latest.sequence,
  )
  const boundedLimit = normalizeLimit(limit)
  if (ordered.length <= boundedLimit) return selection(ordered, ordered.length)

  const chosen = new Map<string, AccessPoint>()
  const key = (point: AccessPoint) =>
    `${point.nodeId}:${point.sequences[0]}:${point.sequences.at(-1)}`
  const add = (point: AccessPoint | undefined) => {
    if (point && chosen.size < boundedLimit) chosen.set(key(point), point)
  }

  add(ordered.find((point) => point.sequences.includes(selectedSequence ?? -1)))
  ordered
    .filter((point) => point.active)
    .reverse()
    .forEach(add)
  const aggregates = ordered.filter((point) => point.aggregate)
  const aggregateTarget = Math.max(chosen.size, Math.floor(boundedLimit * 0.25))
  evenlySpaced(aggregates, aggregateTarget - chosen.size).forEach(add)

  const recentTarget = Math.max(chosen.size, Math.floor(boundedLimit * 0.75))
  for (
    let index = ordered.length - 1;
    index >= 0 && chosen.size < recentTarget;
    index -= 1
  )
    add(ordered[index])

  const remaining = ordered.filter((point) => !chosen.has(key(point)))
  evenlySpaced(remaining, boundedLimit - chosen.size).forEach(add)

  return selection(
    [...chosen.values()].sort(
      (left, right) => left.latest.sequence - right.latest.sequence,
    ),
    ordered.length,
  )
}

function maximumAccess(
  accesses: readonly TrailAccess[],
  value: (access: TrailAccess) => number,
): TrailAccess | undefined {
  return accesses.reduce<TrailAccess | undefined>((maximum, access) => {
    const candidate = Number.isFinite(value(access)) ? value(access) : 0
    const current =
      maximum && Number.isFinite(value(maximum)) ? value(maximum) : 0
    return !maximum || candidate > current ? access : maximum
  }, undefined)
}

function evenlySpaced<T>(items: readonly T[], count: number): T[] {
  if (count <= 0 || items.length === 0) return []
  if (count >= items.length) return [...items]
  if (count === 1) return [items[0]!]
  return Array.from({ length: count }, (_, index) => {
    const itemIndex = Math.floor((index * (items.length - 1)) / (count - 1))
    return items[itemIndex]!
  })
}

function normalizeLimit(limit: number): number {
  return Number.isFinite(limit) ? Math.max(1, Math.floor(limit)) : 1
}

function selection<T>(items: T[], totalCount: number): LodSelection<T> {
  const omittedCount = Math.max(0, totalCount - items.length)
  return { items, totalCount, omittedCount, active: omittedCount > 0 }
}
