import { expect, it } from 'vitest'
import { formatDuration } from '../src/inspector/format'

it('formats reported durations without implying missing values are zero', () => {
  expect(formatDuration(undefined)).toBe('Not reported by agent')
  expect(formatDuration(3_661_000)).toBe('1h 1m 1s')
})
