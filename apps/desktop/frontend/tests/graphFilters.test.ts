import { describe, expect, it } from 'vitest'
import {
  DEFAULT_VISUALIZATION_FILTERS,
  applyVisualizationFilters,
  availableFilterValues,
  hasActiveFilters,
  normalizeVisualizationFilters,
  toggleFilterValue,
} from '../src/filters/graphFilters'
import type { GraphSnapshot, LiveFocusState } from '../src/graph/types'

const graph: GraphSnapshot = {
  nodes: [
    { id: 'root', path: '.', kind: 'root', position: [0, 0, 0] },
    { id: 'src', path: 'src', kind: 'directory', position: [1, 0, 0] },
    { id: 'app', path: 'src/App.tsx', kind: 'source', position: [2, 0, 0] },
    {
      id: 'test',
      path: 'src/App.test.tsx',
      kind: 'test',
      position: [3, 0, 0],
    },
    {
      id: 'doc',
      path: 'docs/README.md',
      kind: 'documentation',
      position: [4, 0, 0],
    },
  ],
  edges: [
    { source: 'root', target: 'src' },
    { source: 'src', target: 'app' },
    { source: 'src', target: 'test' },
    { source: 'root', target: 'doc' },
  ],
}

const focus: LiveFocusState = {
  session_id: 'session',
  active_node_id: 'test',
  active_path: 'src/App.test.tsx',
  previous_node_id: 'app',
  previous_path: 'src/App.tsx',
  secondary_node_ids: ['doc'],
  secondary_paths: ['docs/README.md'],
  operation: 'patch',
  confidence: 'exact',
  trail: [
    {
      sequence: 1,
      node_id: 'app',
      path: 'src/App.tsx',
      started_at: '2026-08-04T10:00:00Z',
      ended_at: '2026-08-04T10:01:00Z',
      duration_ms: 60_000,
      operations: ['read'],
      source: 'codex',
      confidence: 'observed',
      agent_id: 'codex',
    },
    {
      sequence: 2,
      node_id: 'test',
      path: 'src/App.test.tsx',
      started_at: '2026-08-04T11:00:00Z',
      ended_at: '2026-08-04T11:01:00Z',
      duration_ms: 60_000,
      operations: ['patch'],
      source: 'filesystem',
      confidence: 'exact',
      agent_id: 'wrapper',
    },
    {
      sequence: 3,
      node_id: 'doc',
      path: 'docs/README.md',
      started_at: '2026-08-04T11:30:00Z',
      ended_at: '2026-08-04T11:31:00Z',
      duration_ms: 60_000,
      operations: ['delete'],
      source: 'filesystem',
      confidence: 'inferred',
    },
  ],
}

describe('graph filters', () => {
  it('applies path, file type, operation, confidence, agent, and time filters deterministically', () => {
    const result = applyVisualizationFilters(
      graph,
      focus,
      normalizeVisualizationFilters({
        path: 'app',
        fileTypes: ['test'],
        operations: ['patch'],
        confidences: ['exact'],
        agents: ['wrapper'],
        timeRange: 'custom',
        customStart: '2026-08-04T10:30:00Z',
        customEnd: '2026-08-04T11:15:00Z',
      }),
      Date.parse('2026-08-04T12:00:00Z'),
    )

    expect(result.graph.nodes.map((node) => node.id)).toEqual([
      'root',
      'src',
      'test',
    ])
    expect(result.graph.edges).toEqual([
      { source: 'root', target: 'src' },
      { source: 'src', target: 'test' },
    ])
    expect(result.focus?.trail.map((access) => access.sequence)).toEqual([2])
    expect(result.focus?.active_node_id).toBe('test')
    expect(result.focus?.previous_node_id).toBeUndefined()
    expect(result.visibleAccesses).toBe(1)
  })

  it('uses relative time windows without changing exact trail values', () => {
    const result = applyVisualizationFilters(
      graph,
      focus,
      normalizeVisualizationFilters({ timeRange: 'last-hour' }),
      Date.parse('2026-08-04T11:45:00Z'),
    )

    expect(result.focus?.trail.map((access) => access.sequence)).toEqual([2, 3])
    expect(result.focus?.trail[0]?.duration_ms).toBe(60_000)
  })

  it('normalizes options, toggles values, and detects reset state', () => {
    expect(
      normalizeVisualizationFilters({
        operations: ['patch', 'read', 'patch'],
      }).operations,
    ).toEqual(['patch', 'read'])
    expect(toggleFilterValue(['read'], 'patch')).toEqual(['patch', 'read'])
    expect(toggleFilterValue(['patch', 'read'], 'patch')).toEqual(['read'])
    expect(hasActiveFilters(DEFAULT_VISUALIZATION_FILTERS)).toBe(false)
    expect(
      hasActiveFilters({
        ...DEFAULT_VISUALIZATION_FILTERS,
        agents: ['unknown'],
      }),
    ).toBe(true)
  })

  it('derives accessible option values from graph and current focus', () => {
    expect(availableFilterValues(graph, focus)).toEqual({
      fileTypes: ['documentation', 'source', 'test'],
      operations: ['delete', 'patch', 'read'],
      confidences: ['exact', 'inferred', 'observed'],
      agents: ['codex', 'unknown', 'wrapper'],
    })
  })
})
