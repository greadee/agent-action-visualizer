import { describe, expect, it } from 'vitest'
import { adjacentAccessCursor, formatReplayTime } from '../src/replay/timeline'

describe('replay timeline helpers', () => {
  const timeline = [
    {
      index: 0,
      timestamp: '2026-01-01T00:00:00Z',
      event_type: 'session_started',
      is_access: false,
    },
    {
      index: 1,
      timestamp: '2026-01-01T00:00:01Z',
      event_type: 'file_read',
      is_access: true,
    },
    {
      index: 2,
      timestamp: '2026-01-01T00:00:02Z',
      event_type: 'diff_calculated',
      is_access: false,
    },
    {
      index: 3,
      timestamp: '2026-01-01T00:00:03Z',
      event_type: 'file_patched',
      is_access: true,
    },
  ]

  it('moves only between access events and remains bounded', () => {
    expect(adjacentAccessCursor(timeline, 0, 1)).toBe(1)
    expect(adjacentAccessCursor(timeline, 1, 1)).toBe(3)
    expect(adjacentAccessCursor(timeline, 3, 1)).toBe(3)
    expect(adjacentAccessCursor(timeline, 3, -1)).toBe(1)
    expect(adjacentAccessCursor(timeline, 1, -1)).toBe(1)
  })

  it('formats empty and invalid replay times safely', () => {
    expect(formatReplayTime()).toBe('Before first event')
    expect(formatReplayTime('not-a-time')).toBe('Unknown time')
  })
})
