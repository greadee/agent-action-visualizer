import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { useGraphBridge } from '../src/bridge/useGraphBridge'
import type {
  GraphPatch,
  GraphSnapshot,
  LiveFocusState,
} from '../src/graph/types'

afterEach(() => {
  delete window.runtime
  delete window.go
})

it('resynchronizes after a renderer remount and rejects stale patches', async () => {
  const callbacks = new Map<string, (payload: unknown) => void>()
  window.runtime = {
    EventsOnMultiple: (name, callback) => {
      callbacks.set(name, callback)
      return () => callbacks.delete(name)
    },
  }
  const authoritative: GraphSnapshot = {
    revision: 3,
    nodes: [
      {
        id: 'current',
        path: 'current.go',
        kind: 'source',
        position: [1, 0, 0],
      },
    ],
    edges: [],
  }
  const focus: LiveFocusState = {
    session_id: 'session',
    active_node_id: 'current',
    active_path: 'current.go',
    trail: [],
  }
  window.go = {
    main: {
      App: {
        LoadProject: vi.fn(),
        PublishActivityEvent: vi.fn(),
        ListPersistedSessions: vi.fn(),
        ReplayPersistedSession: vi.fn(),
        Resync: vi.fn().mockResolvedValue({ graph: authoritative, focus }),
      },
    },
  }
  const fallback: GraphSnapshot = { revision: 1, nodes: [], edges: [] }
  const { result } = renderHook(() => useGraphBridge(fallback))

  await waitFor(() => expect(result.current.graph.revision).toBe(3))
  expect(result.current.liveFocus?.active_path).toBe('current.go')

  act(() => {
    callbacks.get('aav:graph:patch')?.({
      revision: 2,
      added: [],
      updated: [],
      removed: ['current'],
      edges: [],
    } satisfies GraphPatch)
  })
  expect(result.current.graph).toEqual(authoritative)
})
