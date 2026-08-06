import type { GraphSnapshot, LiveFocusState, TrailAccess } from '../graph/types'

export interface SessionAnalytics {
  totalAccesses: number
  uniqueFiles: number
  totalDurationMs: number
  unknownDurationCount: number
  linesAdded: number
  linesDeleted: number
  unknownWorkCount: number
  binaryWorkCount: number
  unsupportedWorkCount: number
  pendingWorkCount: number
  operationCounts: OperationCounts
  confidenceCounts: AnalyticsBucket[]
  workConfidenceCounts: AnalyticsBucket[]
  fileActivity: AnalyticsBucket[]
  directoryActivity: DirectoryActivity[]
  agentActivity: AgentActivity[]
}

export interface OperationCounts {
  created: number
  modified: number
  deleted: number
  read: number
  other: number
}

export interface AnalyticsBucket {
  label: string
  count: number
}

export interface DirectoryActivity extends AnalyticsBucket {
  durationMs: number
  linesAdded: number
  linesDeleted: number
}

export interface AgentActivity extends AnalyticsBucket {
  durationMs: number
  linesAdded: number
  linesDeleted: number
  unknownWork: number
}

const knownWorkStatuses = new Set(['known', 'empty'])

export function buildSessionAnalytics(
  graph: GraphSnapshot,
  focus?: LiveFocusState,
): SessionAnalytics {
  const visibleFileIds = new Set(
    graph.nodes
      .filter((node) => node.kind !== 'root' && node.kind !== 'directory')
      .map((node) => node.id),
  )
  const trail = (focus?.trail ?? []).filter(
    (access) => !access.node_id || visibleFileIds.has(access.node_id),
  )
  const uniqueFiles = new Set(trail.map((access) => access.path)).size
  const fileCounts = new Map<string, number>()
  const confidenceCounts = new Map<string, number>()
  const workConfidenceCounts = new Map<string, number>()
  const directoryActivity = new Map<string, DirectoryActivity>()
  const agentActivity = new Map<string, AgentActivity>()
  const operationCounts: OperationCounts = {
    created: 0,
    modified: 0,
    deleted: 0,
    read: 0,
    other: 0,
  }
  let totalDurationMs = 0
  let unknownDurationCount = 0
  let linesAdded = 0
  let linesDeleted = 0
  let unknownWorkCount = 0
  let binaryWorkCount = 0
  let unsupportedWorkCount = 0
  let pendingWorkCount = 0

  for (const access of trail) {
    const durationMs = normalizedDuration(access)
    if (durationMs === undefined) unknownDurationCount += 1
    else totalDurationMs += durationMs

    const work = normalizedWork(access)
    if (work.status === 'known') {
      linesAdded += work.linesAdded
      linesDeleted += work.linesDeleted
    } else if (work.status === 'binary') binaryWorkCount += 1
    else if (work.status === 'unsupported_encoding') unsupportedWorkCount += 1
    else if (work.status === 'pending') pendingWorkCount += 1
    else unknownWorkCount += 1

    increment(fileCounts, access.path)
    increment(confidenceCounts, access.confidence || 'unknown')
    increment(workConfidenceCounts, access.work_confidence ?? 'unknown')
    applyOperations(operationCounts, access.operations)

    const directory = parentPath(access.path)
    const directoryBucket = ensureDirectoryBucket(directoryActivity, directory)
    directoryBucket.count += 1
    directoryBucket.durationMs += durationMs ?? 0
    if (work.status === 'known') {
      directoryBucket.linesAdded += work.linesAdded
      directoryBucket.linesDeleted += work.linesDeleted
    }

    const agent = access.agent_id ?? 'unknown'
    const agentBucket = ensureAgentBucket(agentActivity, agent)
    agentBucket.count += 1
    agentBucket.durationMs += durationMs ?? 0
    if (work.status === 'known') {
      agentBucket.linesAdded += work.linesAdded
      agentBucket.linesDeleted += work.linesDeleted
    } else {
      agentBucket.unknownWork += 1
    }
  }

  return {
    totalAccesses: trail.length,
    uniqueFiles,
    totalDurationMs,
    unknownDurationCount,
    linesAdded,
    linesDeleted,
    unknownWorkCount,
    binaryWorkCount,
    unsupportedWorkCount,
    pendingWorkCount,
    operationCounts,
    confidenceCounts: sortedBuckets(confidenceCounts),
    workConfidenceCounts: sortedBuckets(workConfidenceCounts),
    fileActivity: sortedBuckets(fileCounts),
    directoryActivity: sortedComplexBuckets(directoryActivity),
    agentActivity: sortedComplexBuckets(agentActivity),
  }
}

function normalizedDuration(access: TrailAccess) {
  if (!Number.isFinite(access.duration_ms)) return undefined
  return Math.max(0, Math.floor(access.duration_ms))
}

function normalizedWork(access: TrailAccess):
  | { status: 'known'; linesAdded: number; linesDeleted: number }
  | {
      status: 'unknown' | 'binary' | 'unsupported_encoding' | 'pending'
    } {
  if (access.work_status === 'binary') return { status: 'binary' }
  if (access.work_status === 'unsupported_encoding') {
    return { status: 'unsupported_encoding' }
  }
  if (access.work_status === 'pending') return { status: 'pending' }
  if (
    knownWorkStatuses.has(access.work_status ?? '') ||
    access.lines_added !== undefined ||
    access.lines_deleted !== undefined
  ) {
    return {
      status: 'known',
      linesAdded: Math.max(0, Math.floor(access.lines_added ?? 0)),
      linesDeleted: Math.max(0, Math.floor(access.lines_deleted ?? 0)),
    }
  }
  return { status: 'unknown' }
}

function applyOperations(counts: OperationCounts, operations: string[]) {
  const normalized = new Set(
    operations.map((operation) => operation.toLowerCase()),
  )
  if (matchesOperation(normalized, ['create', 'created', 'add', 'new'])) {
    counts.created += 1
  }
  if (
    matchesOperation(normalized, [
      'patch',
      'write',
      'modify',
      'modified',
      'rename',
      'move',
    ])
  ) {
    counts.modified += 1
  }
  if (
    matchesOperation(normalized, ['delete', 'deleted', 'remove', 'removed'])
  ) {
    counts.deleted += 1
  }
  if (matchesOperation(normalized, ['read', 'open', 'inspect'])) {
    counts.read += 1
  }
  if (normalized.size === 0) counts.other += 1
  for (const operation of normalized) {
    if (
      !matchesOperation(new Set([operation]), [
        'create',
        'created',
        'add',
        'new',
        'patch',
        'write',
        'modify',
        'modified',
        'rename',
        'move',
        'delete',
        'deleted',
        'remove',
        'removed',
        'read',
        'open',
        'inspect',
      ])
    ) {
      counts.other += 1
    }
  }
}

function matchesOperation(operations: Set<string>, terms: string[]) {
  return terms.some((term) =>
    Array.from(operations).some((operation) => operation.includes(term)),
  )
}

function ensureDirectoryBucket(
  buckets: Map<string, DirectoryActivity>,
  label: string,
) {
  const existing = buckets.get(label)
  if (existing) return existing
  const next = {
    label,
    count: 0,
    durationMs: 0,
    linesAdded: 0,
    linesDeleted: 0,
  }
  buckets.set(label, next)
  return next
}

function ensureAgentBucket(buckets: Map<string, AgentActivity>, label: string) {
  const existing = buckets.get(label)
  if (existing) return existing
  const next = {
    label,
    count: 0,
    durationMs: 0,
    linesAdded: 0,
    linesDeleted: 0,
    unknownWork: 0,
  }
  buckets.set(label, next)
  return next
}

function increment(values: Map<string, number>, key: string) {
  values.set(key, (values.get(key) ?? 0) + 1)
}

function sortedBuckets(values: Map<string, number>) {
  return Array.from(values.entries())
    .map(([label, count]) => ({ label, count }))
    .sort(compareBuckets)
}

function sortedComplexBuckets<T extends AnalyticsBucket>(
  values: Map<string, T>,
) {
  return Array.from(values.values()).sort(compareBuckets)
}

function compareBuckets(left: AnalyticsBucket, right: AnalyticsBucket) {
  return right.count - left.count || left.label.localeCompare(right.label)
}

function parentPath(path: string) {
  const normalized = path.replaceAll('\\', '/')
  const separator = normalized.lastIndexOf('/')
  return separator < 0 ? '.' : normalized.slice(0, separator)
}
