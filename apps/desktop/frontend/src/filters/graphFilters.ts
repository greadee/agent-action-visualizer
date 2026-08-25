import type {
  GraphNode,
  GraphSnapshot,
  LiveFocusState,
  TrailAccess,
} from '../graph/types'

export type TimeRangeMode = 'all' | 'last-hour' | 'last-day' | 'custom'

export interface VisualizationFilters {
  path: string
  fileTypes: string[]
  operations: string[]
  confidences: string[]
  agents: string[]
  timeRange: TimeRangeMode
  customStart: string
  customEnd: string
}

export const DEFAULT_VISUALIZATION_FILTERS: VisualizationFilters = {
  path: '',
  fileTypes: [],
  operations: [],
  confidences: [],
  agents: [],
  timeRange: 'all',
  customStart: '',
  customEnd: '',
}

export function normalizeVisualizationFilters(
  value: Partial<VisualizationFilters> | undefined,
): VisualizationFilters {
  return {
    ...DEFAULT_VISUALIZATION_FILTERS,
    ...(value ?? {}),
    fileTypes: uniqueSorted(value?.fileTypes ?? []),
    operations: uniqueSorted(value?.operations ?? []),
    confidences: uniqueSorted(value?.confidences ?? []),
    agents: uniqueSorted(value?.agents ?? []),
  }
}

export function applyVisualizationFilters(
  graph: GraphSnapshot,
  focus: LiveFocusState | undefined,
  filters: VisualizationFilters,
  nowMs: number,
): { graph: GraphSnapshot; focus?: LiveFocusState; visibleAccesses: number } {
  const normalized = normalizeVisualizationFilters(filters)
  const timeWindow = resolveTimeWindow(normalized, nowMs)
  const matchingTrail = (focus?.trail ?? []).filter((access) =>
    accessMatches(access, normalized, timeWindow),
  )
  const matchingNodeIds = new Set(
    matchingTrail
      .map((access) => access.node_id)
      .filter((id): id is string => Boolean(id)),
  )
  const visibleNodes = graph.nodes.filter(
    (node) =>
      node.kind === 'root' ||
      node.kind === 'directory' ||
      nodeMatches(node, normalized, matchingNodeIds),
  )
  const visibleIds = new Set(visibleNodes.map((node) => node.id))
  const filteredGraph = {
    ...graph,
    nodes: visibleNodes,
    edges: graph.edges.filter(
      (edge) => visibleIds.has(edge.source) && visibleIds.has(edge.target),
    ),
  }
  if (!focus)
    return { graph: filteredGraph, visibleAccesses: matchingTrail.length }
  const activeVisible = Boolean(
    focus.active_node_id && visibleIds.has(focus.active_node_id),
  )
  const previousVisible = Boolean(
    focus.previous_node_id && visibleIds.has(focus.previous_node_id),
  )
  const secondary = (focus.secondary_node_ids ?? [])
    .map((id, index) => ({ id, path: focus.secondary_paths?.[index] }))
    .filter((item) => visibleIds.has(item.id))
  return {
    graph: filteredGraph,
    visibleAccesses: matchingTrail.length,
    focus: {
      ...focus,
      active_node_id: activeVisible ? focus.active_node_id : undefined,
      active_path: activeVisible ? focus.active_path : undefined,
      previous_node_id: previousVisible ? focus.previous_node_id : undefined,
      previous_path: previousVisible ? focus.previous_path : undefined,
      secondary_node_ids: secondary.map((item) => item.id),
      secondary_paths: secondary
        .map((item) => item.path)
        .filter((path): path is string => Boolean(path)),
      trail: matchingTrail,
    },
  }
}

export function hasActiveFilters(filters: VisualizationFilters) {
  const normalized = normalizeVisualizationFilters(filters)
  return (
    normalized.path.trim() !== '' ||
    normalized.fileTypes.length > 0 ||
    normalized.operations.length > 0 ||
    normalized.confidences.length > 0 ||
    normalized.agents.length > 0 ||
    normalized.timeRange !== 'all' ||
    normalized.customStart !== '' ||
    normalized.customEnd !== ''
  )
}

export function availableFilterValues(
  graph: GraphSnapshot,
  focus: LiveFocusState | undefined,
) {
  return {
    fileTypes: uniqueSorted(
      graph.nodes
        .map((node) => node.kind)
        .filter((kind) => kind !== 'root' && kind !== 'directory'),
    ),
    operations: uniqueSorted(
      (focus?.trail ?? []).flatMap((access) => access.operations),
    ),
    confidences: uniqueSorted(
      (focus?.trail ?? []).map((access) => access.confidence).filter(Boolean),
    ),
    agents: uniqueSorted(
      (focus?.trail ?? [])
        .map((access) => access.agent_id ?? 'unknown')
        .filter(Boolean),
    ),
  }
}

export function toggleFilterValue(values: string[], value: string) {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : uniqueSorted([...values, value])
}

function nodeMatches(
  node: GraphNode,
  filters: VisualizationFilters,
  matchingNodeIds: Set<string>,
) {
  if (filters.path && !pathMatches(node.path, filters.path)) return false
  if (filters.fileTypes.length > 0 && !filters.fileTypes.includes(node.kind)) {
    return false
  }
  if (requiresAccessEvidence(filters) && !matchingNodeIds.has(node.id)) {
    return false
  }
  return true
}

function accessMatches(
  access: TrailAccess,
  filters: VisualizationFilters,
  timeWindow: { start?: number; end?: number },
) {
  if (filters.path && !pathMatches(access.path, filters.path)) return false
  if (
    filters.operations.length > 0 &&
    !access.operations.some((operation) =>
      filters.operations.includes(operation),
    )
  ) {
    return false
  }
  if (
    filters.confidences.length > 0 &&
    !filters.confidences.includes(access.confidence)
  ) {
    return false
  }
  const agent = access.agent_id ?? 'unknown'
  if (filters.agents.length > 0 && !filters.agents.includes(agent)) {
    return false
  }
  const started = Date.parse(access.started_at)
  if (Number.isNaN(started)) return false
  if (timeWindow.start !== undefined && started < timeWindow.start) return false
  if (timeWindow.end !== undefined && started > timeWindow.end) return false
  return true
}

function resolveTimeWindow(filters: VisualizationFilters, nowMs: number) {
  if (filters.timeRange === 'last-hour') {
    return { start: nowMs - 60 * 60 * 1000 }
  }
  if (filters.timeRange === 'last-day') {
    return { start: nowMs - 24 * 60 * 60 * 1000 }
  }
  if (filters.timeRange === 'custom') {
    return {
      start: parseDateTimeLocal(filters.customStart),
      end: parseDateTimeLocal(filters.customEnd),
    }
  }
  return {}
}

function requiresAccessEvidence(filters: VisualizationFilters) {
  return (
    filters.operations.length > 0 ||
    filters.confidences.length > 0 ||
    filters.agents.length > 0 ||
    filters.timeRange !== 'all' ||
    filters.customStart !== '' ||
    filters.customEnd !== ''
  )
}

function pathMatches(path: string, query: string) {
  return path.toLowerCase().includes(query.trim().toLowerCase())
}

function parseDateTimeLocal(value: string) {
  if (!value) return undefined
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? undefined : parsed
}

function uniqueSorted(values: string[]) {
  return Array.from(new Set(values.filter(Boolean))).sort((left, right) =>
    left.localeCompare(right),
  )
}
