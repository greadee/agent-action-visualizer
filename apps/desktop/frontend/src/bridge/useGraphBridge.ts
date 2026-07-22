import { useEffect, useState } from 'react'
import type { GraphPatch, GraphSnapshot } from '../graph/types'
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
        }
      }
    }
  }
}

export function useGraphBridge(fallback: GraphSnapshot) {
  const [graph, setGraph] = useState(fallback)
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
    return () => {
      offSnapshot?.()
      offPatch?.()
    }
  }, [])

  async function loadProject(root: string) {
    const load = window.go?.main?.App?.LoadProject
    if (!load)
      throw new Error('Desktop bridge is unavailable in browser preview')
    setGraph(await load(root))
  }
  return { graph, loadProject }
}
