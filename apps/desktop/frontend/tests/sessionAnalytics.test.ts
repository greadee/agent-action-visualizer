import { describe, expect, it } from 'vitest'
import { buildSessionAnalytics } from '../src/analytics/sessionAnalytics'
import { applyVisualizationFilters } from '../src/filters/graphFilters'
import type { GraphSnapshot, LiveFocusState } from '../src/graph/types'

const graph: GraphSnapshot = {
  nodes: [
    { id: 'root', path: '.', kind: 'root', position: [0, 0, 0] },
    { id: 'src', path: 'src', kind: 'directory', position: [1, 0, 0] },
    { id: 'app', path: 'src/App.tsx', kind: 'source', position: [2, 0, 0] },
    { id: 'test', path: 'src/App.test.tsx', kind: 'test', position: [3, 0, 0] },
    { id: 'bin', path: 'assets/icon.bin', kind: 'asset', position: [4, 0, 0] },
  ],
  edges: [
    { source: 'root', target: 'src' },
    { source: 'src', target: 'app' },
    { source: 'src', target: 'test' },
    { source: 'root', target: 'bin' },
  ],
}

const focus: LiveFocusState = {
  session_id: 'replay-session',
  active_node_id: 'test',
  active_path: 'src/App.test.tsx',
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
      operations: ['create'],
      source: 'codex',
      confidence: 'exact',
      agent_id: 'codex',
      lines_added: 12,
      lines_deleted: 0,
      work_status: 'known',
      work_source: 'structured_patch',
      work_confidence: 'exact',
    },
    {
      sequence: 2,
      node_id: 'test',
      path: 'src/App.test.tsx',
      started_at: '2026-08-04T10:02:00Z',
      ended_at: '2026-08-04T10:03:30Z',
      duration_ms: 90_000,
      operations: ['patch', 'read'],
      source: 'filesystem',
      confidence: 'observed',
      agent_id: 'wrapper',
      lines_added: 4,
      lines_deleted: 2,
      work_status: 'known',
      work_source: 'git_numstat',
      work_confidence: 'observed',
    },
    {
      sequence: 3,
      node_id: 'app',
      path: 'src/App.tsx',
      started_at: '2026-08-04T10:04:00Z',
      ended_at: '2026-08-04T10:04:30Z',
      duration_ms: 30_000,
      operations: ['delete'],
      source: 'filesystem',
      confidence: 'inferred',
      work_status: 'unknown',
      work_source: 'unknown',
      work_confidence: 'inferred',
    },
    {
      sequence: 4,
      node_id: 'bin',
      path: 'assets/icon.bin',
      started_at: '2026-08-04T10:05:00Z',
      ended_at: '2026-08-04T10:05:01Z',
      duration_ms: 1_000,
      operations: ['patch'],
      source: 'filesystem',
      confidence: 'exact',
      agent_id: 'wrapper',
      work_status: 'binary',
      work_source: 'unknown',
      work_confidence: 'unknown',
    },
  ],
}

describe('session analytics', () => {
  it('reconciles access, duration, work, operation, confidence, directory, and agent totals from source records', () => {
    const analytics = buildSessionAnalytics(graph, focus)

    expect(analytics.totalAccesses).toBe(4)
    expect(analytics.uniqueFiles).toBe(3)
    expect(analytics.totalDurationMs).toBe(181_000)
    expect(analytics.linesAdded).toBe(16)
    expect(analytics.linesDeleted).toBe(2)
    expect(analytics.unknownWorkCount).toBe(1)
    expect(analytics.binaryWorkCount).toBe(1)
    expect(analytics.operationCounts).toEqual({
      created: 1,
      modified: 2,
      deleted: 1,
      read: 1,
      other: 0,
    })
    expect(analytics.confidenceCounts).toEqual([
      { label: 'exact', count: 2 },
      { label: 'inferred', count: 1 },
      { label: 'observed', count: 1 },
    ])
    expect(analytics.directoryActivity[0]).toEqual({
      label: 'src',
      count: 3,
      durationMs: 180_000,
      linesAdded: 16,
      linesDeleted: 2,
    })
    expect(analytics.agentActivity).toEqual([
      {
        label: 'wrapper',
        count: 2,
        durationMs: 91_000,
        linesAdded: 4,
        linesDeleted: 2,
        unknownWork: 1,
      },
      {
        label: 'codex',
        count: 1,
        durationMs: 60_000,
        linesAdded: 12,
        linesDeleted: 0,
        unknownWork: 0,
      },
      {
        label: 'unknown',
        count: 1,
        durationMs: 30_000,
        linesAdded: 0,
        linesDeleted: 0,
        unknownWork: 1,
      },
    ])
  })

  it('uses the filtered graph and trail without changing exact values', () => {
    const filtered = applyVisualizationFilters(
      graph,
      focus,
      {
        path: 'app',
        fileTypes: ['test'],
        operations: ['patch'],
        confidences: [],
        agents: ['wrapper'],
        timeRange: 'all',
        customStart: '',
        customEnd: '',
      },
      Date.parse('2026-08-04T11:00:00Z'),
    )

    const analytics = buildSessionAnalytics(filtered.graph, filtered.focus)

    expect(analytics.totalAccesses).toBe(1)
    expect(analytics.uniqueFiles).toBe(1)
    expect(analytics.totalDurationMs).toBe(90_000)
    expect(analytics.linesAdded).toBe(4)
    expect(analytics.linesDeleted).toBe(2)
    expect(analytics.fileActivity).toEqual([
      { label: 'src/App.test.tsx', count: 1 },
    ])
  })

  it('marks unsupported, pending, empty, and absent duration states explicitly', () => {
    const analytics = buildSessionAnalytics(graph, {
      session_id: 'edge-session',
      trail: [
        {
          sequence: 1,
          node_id: 'app',
          path: 'src/App.tsx',
          started_at: '2026-08-04T10:00:00Z',
          duration_ms: Number.NaN,
          operations: [],
          source: 'codex',
          confidence: '',
          work_status: 'unsupported_encoding',
        },
        {
          sequence: 2,
          node_id: 'test',
          path: 'src/App.test.tsx',
          started_at: '2026-08-04T10:01:00Z',
          duration_ms: 0,
          operations: ['patch'],
          source: 'filesystem',
          confidence: 'exact',
          lines_added: 0,
          lines_deleted: 0,
          work_status: 'empty',
        },
        {
          sequence: 3,
          node_id: 'test',
          path: 'src/App.test.tsx',
          started_at: '2026-08-04T10:02:00Z',
          duration_ms: 10,
          operations: ['patch'],
          source: 'filesystem',
          confidence: 'exact',
          work_status: 'pending',
        },
      ],
    })

    expect(analytics.unknownDurationCount).toBe(1)
    expect(analytics.unsupportedWorkCount).toBe(1)
    expect(analytics.pendingWorkCount).toBe(1)
    expect(analytics.unknownWorkCount).toBe(0)
    expect(analytics.linesAdded).toBe(0)
    expect(analytics.linesDeleted).toBe(0)
    expect(analytics.confidenceCounts).toContainEqual({
      label: 'unknown',
      count: 1,
    })
  })
})
