export interface RenderDiagnosticsSnapshot {
  fps: number
  cpuFrameP95Ms: number
  gpuFrameP95Ms?: number
  drawCalls: number
  triangles: number
  lines: number
  points: number
  geometries: number
  textures: number
  jsHeapMb?: number
}

export function percentile95(samples: readonly number[]): number {
  if (samples.length === 0) return 0
  const ordered = [...samples].sort((left, right) => left - right)
  return ordered[Math.ceil(ordered.length * 0.95) - 1] ?? 0
}

export function framesPerSecond(samples: readonly number[]): number {
  if (samples.length === 0) return 0
  const totalMs = samples.reduce((total, sample) => total + sample, 0)
  return totalMs <= 0 ? 0 : (samples.length * 1_000) / totalMs
}

export function formatDiagnostic(value: number, digits = 1): string {
  return Number.isFinite(value) ? value.toFixed(digits) : 'unavailable'
}
