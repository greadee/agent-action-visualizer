import { useEffect, useRef, useState } from 'react'
import type {
  ActivityEvent,
  GraphPatch,
  GraphSnapshot,
  LiveFocusState,
  PersistedSessionSummary,
  ReplaySession,
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
          ListPersistedSessions: () => Promise<PersistedSessionSummary[]>
          ReplayPersistedSession: (
            sessionID: string,
            cursor: number,
          ) => Promise<ReplaySession>
        }
      }
    }
  }
}

export function useGraphBridge(
  fallback: GraphSnapshot,
  fallbackFocus?: LiveFocusState,
) {
  const [graph, setGraph] = useState(fallback)
  const [liveFocus, setLiveFocus] = useState<LiveFocusState | undefined>(
    fallbackFocus,
  )
  const fallbackHistory = useRef<
    Map<string, { event: ActivityEvent; focus: LiveFocusState }[]>
  >(new Map())
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
        const next = { session_id: event.session_id, trail: [] }
        recordFallbackEvent(fallbackHistory.current, event, next)
        return next
      }
      if (!event.path) {
        const next = current ?? { session_id: event.session_id, trail: [] }
        recordFallbackEvent(fallbackHistory.current, event, next)
        return next
      }
      const active = graph.nodes.find((node) => node.path === event.path)
      const secondaryPaths = event.metadata?.secondary_paths ?? []
      const previousTrail = current?.trail ?? []
      const lastAccess = previousTrail.at(-1)
      const sameAccess = lastAccess?.path === event.path
      const closedTrail = sameAccess
        ? previousTrail.slice(0, -1)
        : previousTrail.map((access, index) =>
            index === previousTrail.length - 1 && !access.ended_at
              ? {
                  ...access,
                  ended_at: event.timestamp,
                  duration_ms: Math.max(
                    0,
                    Date.parse(event.timestamp) - Date.parse(access.started_at),
                  ),
                }
              : access,
          )
      const operations = Array.from(
        new Set(
          sameAccess
            ? [...(lastAccess?.operations ?? []), event.operation].filter(
                (operation): operation is string => Boolean(operation),
              )
            : event.operation
              ? [event.operation]
              : [],
        ),
      )
      const hasStructuredDelta =
        event.lines_added !== undefined || event.lines_deleted !== undefined
      const linesAdded = hasStructuredDelta
        ? Math.max(0, event.lines_added ?? 0)
        : sameAccess
          ? lastAccess?.lines_added
          : undefined
      const linesDeleted = hasStructuredDelta
        ? Math.max(0, event.lines_deleted ?? 0)
        : sameAccess
          ? lastAccess?.lines_deleted
          : undefined
      const workStatus = event.is_binary
        ? ('binary' as const)
        : hasStructuredDelta
          ? linesAdded === 0 && linesDeleted === 0
            ? ('empty' as const)
            : ('known' as const)
          : sameAccess
            ? (lastAccess?.work_status ?? ('unknown' as const))
            : ('unknown' as const)
      const nextAccess = {
        sequence: sameAccess
          ? (lastAccess?.sequence ?? previousTrail.length + 1)
          : previousTrail.length + 1,
        node_id: active?.id,
        path: event.path ?? '',
        started_at: sameAccess
          ? (lastAccess?.started_at ?? event.timestamp)
          : event.timestamp,
        duration_ms: 0,
        operations,
        source: event.source_type,
        confidence: event.source_confidence,
        lines_added: linesAdded,
        lines_deleted: linesDeleted,
        work_status: workStatus,
        work_source: hasStructuredDelta
          ? ('structured_patch' as const)
          : ('unknown' as const),
        work_confidence: hasStructuredDelta
          ? event.source_confidence
          : 'inferred',
      }
      const next = {
        session_id: event.session_id,
        active_node_id: active?.id,
        active_path: event.path,
        previous_node_id: sameAccess
          ? current?.previous_node_id
          : current?.active_node_id,
        previous_path: sameAccess
          ? current?.previous_path
          : current?.active_path,
        secondary_node_ids: secondaryPaths
          .map((path) => graph.nodes.find((node) => node.path === path)?.id)
          .filter((id): id is string => Boolean(id) && id !== active?.id),
        secondary_paths: secondaryPaths,
        operation: event.operation,
        source: event.source_type,
        confidence: event.source_confidence,
        timestamp: event.timestamp,
        trail: [...closedTrail, nextAccess],
      }
      recordFallbackEvent(fallbackHistory.current, event, next)
      return next
    })
  }
  async function listPersistedSessions() {
    const list = window.go?.main?.App?.ListPersistedSessions
    if (list) return list()
    return Array.from(fallbackHistory.current.entries()).map(([id, items]) => ({
      id,
      started_at: items[0]?.event.timestamp ?? '',
      status: 'preview',
    }))
  }

  async function replayPersistedSession(sessionID: string, cursor: number) {
    const replay = window.go?.main?.App?.ReplayPersistedSession
    if (replay) return replay(sessionID, cursor)
    const items = fallbackHistory.current.get(sessionID) ?? []
    if (items.length === 0) throw new Error('No replay fixture is available')
    const index = Math.min(Math.max(0, cursor), items.length - 1)
    const cursorAt = items[index]?.event.timestamp
    return {
      session: {
        id: sessionID,
        started_at: items[0]?.event.timestamp ?? '',
        status: 'preview',
      },
      cursor: index,
      event_count: items.length,
      cursor_at: cursorAt,
      focus: freezePreviewFocus(items[index]?.focus, cursorAt),
      timeline: items.map((item, eventIndex) => ({
        index: eventIndex,
        timestamp: item.event.timestamp,
        event_type: item.event.event_type,
        path: item.event.path,
        is_access: Boolean(item.event.path),
      })),
    }
  }

  return {
    graph,
    liveFocus,
    loadProject,
    publishActivityEvent,
    listPersistedSessions,
    replayPersistedSession,
  }
}

function recordFallbackEvent(
  history: Map<string, { event: ActivityEvent; focus: LiveFocusState }[]>,
  event: ActivityEvent,
  focus: LiveFocusState,
) {
  const events = history.get(event.session_id) ?? []
  if (events.some((item) => item.event.event_id === event.event_id)) return
  history.set(event.session_id, [...events, { event, focus }])
}

function freezePreviewFocus(focus: LiveFocusState | undefined, at?: string) {
  if (!focus || !at) return focus ?? { session_id: '', trail: [] }
  return {
    ...focus,
    trail: focus.trail.map((access) =>
      access.ended_at
        ? access
        : {
            ...access,
            ended_at: at,
            duration_ms: Math.max(
              0,
              Date.parse(at) - Date.parse(access.started_at),
            ),
          },
    ),
  }
}
