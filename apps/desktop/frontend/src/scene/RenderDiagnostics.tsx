import { Html } from '@react-three/drei'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useRef, useState } from 'react'
import {
  formatDiagnostic,
  framesPerSecond,
  percentile95,
  type RenderDiagnosticsSnapshot,
} from '../rendering/diagnostics'

interface TimerQueryExtension {
  TIME_ELAPSED_EXT: number
  GPU_DISJOINT_EXT: number
}

interface PerformanceWithMemory extends Performance {
  memory?: { usedJSHeapSize: number }
}

const emptySnapshot: RenderDiagnosticsSnapshot = {
  fps: 0,
  cpuFrameP95Ms: 0,
  drawCalls: 0,
  triangles: 0,
  lines: 0,
  points: 0,
  geometries: 0,
  textures: 0,
}

export function RenderDiagnostics({ enabled }: { enabled: boolean }) {
  const renderer = useThree((state) => state.gl)
  const frameIntervals = useRef<number[]>([])
  const cpuSamples = useRef<number[]>([])
  const gpuSamples = useRef<number[]>([])
  const lastReportMs = useRef(0)
  const pendingQueries = useRef<WebGLQuery[]>([])
  const timerExtension = useRef<TimerQueryExtension | null | undefined>(
    undefined,
  )
  const [snapshot, setSnapshot] = useState(emptySnapshot)

  useEffect(() => {
    if (enabled) return
    frameIntervals.current = []
    cpuSamples.current = []
    gpuSamples.current = []
    setSnapshot(emptySnapshot)
  }, [enabled])

  useEffect(
    () => () => {
      const context = webGl2Context(renderer.getContext())
      if (!context) return
      pendingQueries.current.forEach((query) => context.deleteQuery(query))
      pendingQueries.current = []
    },
    [renderer],
  )

  useFrame((state, delta) => {
    const context = webGl2Context(state.gl.getContext())
    const extension = context ? resolveTimerExtension(context) : undefined
    pollGpuQueries(
      context,
      extension,
      pendingQueries.current,
      gpuSamples.current,
    )

    let query: WebGLQuery | undefined
    if (enabled && context && extension && pendingQueries.current.length < 4) {
      query = context.createQuery() ?? undefined
      if (query) context.beginQuery(extension.TIME_ELAPSED_EXT, query)
    }

    const startedAt = performance.now()
    state.gl.render(state.scene, state.camera)
    const cpuFrameMs = performance.now() - startedAt

    if (query && context && extension) {
      context.endQuery(extension.TIME_ELAPSED_EXT)
      pendingQueries.current.push(query)
    }
    if (!enabled) return

    frameIntervals.current.push(delta * 1_000)
    cpuSamples.current.push(cpuFrameMs)
    const now = performance.now()
    if (lastReportMs.current === 0) lastReportMs.current = now
    if (now - lastReportMs.current < 1_000) return

    const memory = (performance as PerformanceWithMemory).memory
    setSnapshot({
      fps: framesPerSecond(frameIntervals.current),
      cpuFrameP95Ms: percentile95(cpuSamples.current),
      gpuFrameP95Ms:
        gpuSamples.current.length > 0
          ? percentile95(gpuSamples.current)
          : undefined,
      drawCalls: state.gl.info.render.calls,
      triangles: state.gl.info.render.triangles,
      lines: state.gl.info.render.lines,
      points: state.gl.info.render.points,
      geometries: state.gl.info.memory.geometries,
      textures: state.gl.info.memory.textures,
      jsHeapMb: memory?.usedJSHeapSize
        ? memory.usedJSHeapSize / (1024 * 1024)
        : undefined,
    })
    frameIntervals.current = []
    cpuSamples.current = []
    gpuSamples.current = []
    lastReportMs.current = now
  }, 1)

  if (!enabled) return null
  return (
    <Html fullscreen zIndexRange={[40, 0]}>
      <section className="render-diagnostics" aria-label="Render diagnostics">
        <strong>RENDER HEALTH</strong>
        <span>{formatDiagnostic(snapshot.fps)} fps</span>
        <span>CPU p95 {formatDiagnostic(snapshot.cpuFrameP95Ms, 2)} ms</span>
        <span>
          GPU p95{' '}
          {snapshot.gpuFrameP95Ms === undefined
            ? 'unsupported'
            : `${formatDiagnostic(snapshot.gpuFrameP95Ms, 2)} ms`}
        </span>
        <span>{snapshot.drawCalls} draw calls</span>
        <span>{snapshot.triangles.toLocaleString()} triangles</span>
        <span>{snapshot.lines.toLocaleString()} lines</span>
        <span>
          {snapshot.geometries} geometries / {snapshot.textures} textures
        </span>
        <span>
          JS heap{' '}
          {snapshot.jsHeapMb === undefined
            ? 'unsupported'
            : `${formatDiagnostic(snapshot.jsHeapMb)} MB`}
        </span>
      </section>
    </Html>
  )

  function resolveTimerExtension(context: WebGL2RenderingContext) {
    if (timerExtension.current === undefined) {
      timerExtension.current = context.getExtension(
        'EXT_disjoint_timer_query_webgl2',
      ) as TimerQueryExtension | null
    }
    return timerExtension.current ?? undefined
  }
}

function webGl2Context(
  context: WebGLRenderingContext | WebGL2RenderingContext,
): WebGL2RenderingContext | undefined {
  return typeof WebGL2RenderingContext !== 'undefined' &&
    context instanceof WebGL2RenderingContext
    ? context
    : undefined
}

function pollGpuQueries(
  context: WebGL2RenderingContext | undefined,
  extension: TimerQueryExtension | undefined,
  pending: WebGLQuery[],
  samples: number[],
) {
  if (!context || !extension || pending.length === 0) return
  const query = pending[0]!
  const available = context.getQueryParameter(
    query,
    context.QUERY_RESULT_AVAILABLE,
  ) as boolean
  const disjoint = context.getParameter(extension.GPU_DISJOINT_EXT) as boolean
  if (!available && !disjoint) return
  pending.shift()
  if (available && !disjoint) {
    const nanoseconds = context.getQueryParameter(query, context.QUERY_RESULT)
    if (typeof nanoseconds === 'number' && Number.isFinite(nanoseconds))
      samples.push(nanoseconds / 1_000_000)
  }
  context.deleteQuery(query)
}
