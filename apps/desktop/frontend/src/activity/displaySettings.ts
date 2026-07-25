import type { ActivityMode } from './extrusions'
import type { DurationScale } from './timeExtrusions'

export const DURATION_VISUAL_CAPS_MS = [15_000, 60_000, 120_000] as const
export const WORK_VISUAL_CAPS_LINES = [100, 1_000, 10_000] as const

export interface ActivityDisplaySettings {
  scale: DurationScale
  durationCapMs: number
  workCapLines: number
}

export const DEFAULT_ACTIVITY_DISPLAY_SETTINGS: ActivityDisplaySettings = {
  scale: 'log',
  durationCapMs: 60_000,
  workCapLines: 1_000,
}

export function activityVisualCap(
  mode: ActivityMode,
  settings: ActivityDisplaySettings,
): number {
  return mode === 'time' ? settings.durationCapMs : settings.workCapLines
}

export function formatVisualCap(mode: ActivityMode, value: number): string {
  if (mode === 'work') return `${value.toLocaleString()} lines`
  if (value % 60_000 === 0) return `${value / 60_000} min`
  return `${value / 1_000} sec`
}
