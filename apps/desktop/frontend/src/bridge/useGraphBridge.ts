import { useEffect, useState } from 'react'
import type {
  ActivityEvent,
  GraphPatch,
  GraphSnapshot,
  LiveFocusState,
} from '../graph/types'
import { applyGraphPatch } from '../graph/patch'

declare global {
  interface Window {
    runtime?: {
      EventsOnMultiple: (
        name: string,
        callback: (payload: unknown) => void,
        maxCallbacks: number,
      ) => () => void
    }
    go?: {
      main?: {
        App?: {
          LoadProject: (root: string) => Promise<GraphSnapshot>
          PublishActivityEvent: (
            event: ActivityEvent,
          ) => Promise<LiveFocusState>
        }
      }
    }
  }
}

export function useGraphBridge(fallback: GraphSnapshot) {
  const [graph, setGraph] = useState(fallback)
  const [liveFocus, setLiveFocus] = useState<LiveFocusState>()
  useEffect(() => {
    const events = window.runtime?.EventsOnMultiple
    if (!events) return
    const offSnapshot = events(
      'aav:graph:snapshot',
      (payload) => setGraph(payload as GraphSnapshot),
      -1,
    )
    const offPatch = events(
      'aav:graph:patch',
      (payload) =>
        setGraph((current) => applyGraphPatch(current, payload as GraphPatch)),
      -1,
    )
    const offFocus = events(
      'aav:focus',
      (payload) => setLiveFocus(payload as LiveFocusState),
      -1,
    )
    return () => {
      offSnapshot?.()
      offPatch?.()
      offFocus?.()
    }
  }, [])

  async function loadProject(root: string) {
    const load = window.go?.main?.App?.LoadProject
    if (!load)
      throw new Error('Desktop bridge is unavailable in browser preview')
    setGraph(await load(root))
  }

  async function publishActivityEvent(event: ActivityEvent) {
    const publish = window.go?.main?.App?.PublishActivityEvent
    if (publish) {
      const next = await publish(event)
      setLiveFocus(next)
      return
    }
    setLiveFocus((current) => {
      if (event.event_type === 'session_started') {
        return { session_id: event.session_id }
      }
      const active = graph.nodes.find((node) => node.path === event.path)
      const secondaryPaths = event.metadata?.secondary_paths ?? []
      return {
        session_id: event.session_id,
        active_node_id: active?.id,
        active_path: event.path,
        previous_node_id: current?.active_node_id,
        previous_path: current?.active_path,
        secondary_node_ids: secondaryPaths
          .map((path) => graph.nodes.find((node) => node.path === path)?.id)
          .filter((id): id is string => Boolean(id) && id !== active?.id),
        secondary_paths: secondaryPaths,
        operation: event.operation,
        source: event.source_type,
        confidence: event.source_confidence,
        timestamp: event.timestamp,
      }
    })
  }
  return { graph, liveFocus, loadProject, publishActivityEvent }
}
