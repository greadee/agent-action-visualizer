import type {
  GraphNode,
  GraphSnapshot,
  LiveFocusState,
  TrailAccess,
  Vec3,
} from '../graph/types'

export const SCALE_NODE_COUNTS = [100, 1_000, 5_000, 10_000, 20_000] as const
export const MAX_SCALE_HISTORY = 100_000

const fixtureEpoch = Date.parse('2026-01-01T00:00:00Z')
const accessStepMs = 120_000
const goldenAngle = Math.PI * (3 - Math.sqrt(5))
const fileKinds: GraphNode['kind'][] = [
  'source',
  'source',
  'test',
  'config',
  'documentation',
  'asset',
]
const extensions = ['ts', 'tsx', 'go', 'json', 'md', 'css']

export interface ScaleFixture {
  graph: GraphSnapshot
  focus: LiveFocusState
  nodeCount: number
  accessCount: number
  diagnostics: boolean
}

export function scaleFixtureFromSearch(
  search: string,
  enabled: boolean,
): ScaleFixture | undefined {
  if (!enabled) return undefined
  const parameters = new URLSearchParams(search)
  const nodeCount = Number(parameters.get('scale'))
  if (!SCALE_NODE_COUNTS.some((value) => value === nodeCount)) return undefined

  const requestedHistory = parameters.get('history')
  const accessCount =
    requestedHistory === null
      ? Math.min(MAX_SCALE_HISTORY, nodeCount * 2)
      : clampInteger(Number(requestedHistory), 0, MAX_SCALE_HISTORY)
  return {
    ...createScaleFixture({ nodeCount, accessCount }),
    diagnostics: parameters.get('diagnostics') !== '0',
  }
}

export function createScaleFixture({
  nodeCount,
  accessCount,
}: {
  nodeCount: number
  accessCount: number
}): Omit<ScaleFixture, 'diagnostics'> {
  const boundedNodeCount = clampInteger(nodeCount, 3, 20_000)
  const boundedAccessCount = clampInteger(accessCount, 0, MAX_SCALE_HISTORY)
  const directoryCount = Math.min(
    100,
    Math.max(1, Math.floor((boundedNodeCount - 1) / 200)),
  )
  const fileCount = boundedNodeCount - directoryCount - 1
  const root: GraphNode = {
    id: 'root',
    path: '.',
    kind: 'root',
    position: [0, 0, 0],
  }
  const directories = Array.from({ length: directoryCount }, (_, index) => ({
    id: `scale-dir-${index}`,
    path: `scale/group-${index.toString().padStart(3, '0')}`,
    kind: 'directory' as const,
    position: spherePosition(index, directoryCount, 6.6),
  }))
  const files = Array.from({ length: fileCount }, (_, index) => {
    const directory = directories[index % directoryCount]!
    const extension = extensions[index % extensions.length]!
    return {
      id: `scale-file-${index}`,
      path: `${directory.path}/file-${index.toString().padStart(5, '0')}.${extension}`,
      kind: fileKinds[index % fileKinds.length]!,
      position: spherePosition(index, fileCount, 7.4 + (index % 5) * 0.08),
      activity: {
        group_key: directory.id,
        confidence: 'synthetic',
      },
    }
  })
  const graph: GraphSnapshot = {
    revision: 1,
    nodes: [root, ...directories, ...files],
    edges: [
      ...directories.map((directory) => ({
        source: root.id,
        target: directory.id,
      })),
      ...files.map((file, index) => ({
        source: directories[index % directoryCount]!.id,
        target: file.id,
      })),
    ],
  }
  const trail = Array.from({ length: boundedAccessCount }, (_, index) =>
    scaleAccess(index, boundedAccessCount, files),
  )
  const active = trail.at(-1)
  const previous = trail.at(-2)
  const focus: LiveFocusState = {
    session_id: `scale-${boundedNodeCount}-${boundedAccessCount}`,
    active_node_id: active?.node_id,
    active_path: active?.path,
    previous_node_id: previous?.node_id,
    previous_path: previous?.path,
    operation: active?.operations.at(-1),
    source: 'scale_fixture',
    confidence: 'exact',
    timestamp: active?.started_at,
    trail,
  }
  return {
    graph,
    focus,
    nodeCount: boundedNodeCount,
    accessCount: trail.length,
  }
}

function scaleAccess(
  index: number,
  accessCount: number,
  files: readonly GraphNode[],
): TrailAccess {
  const file = files[index % files.length]!
  const startedAt = fixtureEpoch + index * accessStepMs
  const durationMs = 250 + (index % 120) * 1_000
  const binary = index > 0 && index % 97 === 0
  const unsupported = !binary && index > 0 && index % 89 === 0
  const unknown = !binary && !unsupported && index > 0 && index % 83 === 0
  const known = !binary && !unsupported && !unknown
  const active = index === accessCount - 1
  return {
    sequence: index + 1,
    node_id: file.id,
    path: file.path,
    started_at: new Date(startedAt).toISOString(),
    ended_at: active
      ? undefined
      : new Date(startedAt + durationMs).toISOString(),
    duration_ms: durationMs,
    operations: [index % 3 === 0 ? 'patch' : 'read'],
    source: 'scale_fixture',
    confidence: 'exact',
    agent_id: `scale-agent-${index % 8}`,
    lines_added: known ? index % 500 : undefined,
    lines_deleted: known ? index % 125 : undefined,
    work_status: binary
      ? 'binary'
      : unsupported
        ? 'unsupported_encoding'
        : unknown
          ? 'unknown'
          : 'known',
    work_source: known ? 'structured_patch' : 'unknown',
    work_confidence: known ? 'exact' : 'inferred',
  }
}

function spherePosition(index: number, count: number, radius: number): Vec3 {
  if (count <= 1) return [radius, 0, 0]
  const y = 1 - ((index + 0.5) / count) * 2
  const horizontal = Math.sqrt(Math.max(0, 1 - y * y))
  const angle = index * goldenAngle
  return [
    radius * horizontal * Math.cos(angle),
    radius * y,
    radius * horizontal * Math.sin(angle),
  ]
}

function clampInteger(value: number, minimum: number, maximum: number) {
  if (!Number.isFinite(value)) return minimum
  return Math.min(maximum, Math.max(minimum, Math.floor(value)))
}
