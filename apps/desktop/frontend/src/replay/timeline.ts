import type { ReplayTimelineEntry } from '../graph/types'

export function adjacentAccessCursor(
  timeline: ReplayTimelineEntry[],
  cursor: number,
  direction: -1 | 1,
) {
  const start = direction > 0 ? cursor + 1 : cursor - 1
  for (
    let index = start;
    index >= 0 && index < timeline.length;
    index += direction
  ) {
    if (timeline[index]?.is_access) return index
  }
  return cursor
}

export function formatReplayTime(timestamp?: string) {
  if (!timestamp) return 'Before first event'
  const value = new Date(timestamp)
  return Number.isNaN(value.getTime())
    ? 'Unknown time'
    : value.toLocaleString(undefined, {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
}
