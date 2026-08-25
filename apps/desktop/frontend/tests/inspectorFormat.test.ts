import { expect, it } from 'vitest'
import {
  formatDuration,
  formatExactDuration,
  formatWorkDelta,
} from '../src/inspector/format'

it('formats reported durations without implying missing values are zero', () => {
  expect(formatDuration(undefined)).toBe('Not reported by agent')
  expect(formatDuration(3_661_000)).toBe('1h 1m 1s')
  expect(formatExactDuration(3_661_123)).toBe('1h 1m 1s 123ms')
  expect(formatWorkDelta(12, 4)).toBe('+12 / -4')
  expect(formatWorkDelta(undefined, undefined)).toBe('Work delta unknown')
})
