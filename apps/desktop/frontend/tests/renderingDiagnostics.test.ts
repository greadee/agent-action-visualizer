import { describe, expect, it, vi } from 'vitest'
import { createLineGeometry } from '../src/rendering/lineGeometry'
import {
  formatDiagnostic,
  framesPerSecond,
  percentile95,
} from '../src/rendering/diagnostics'

describe('render diagnostics', () => {
  it('calculates stable frame statistics', () => {
    expect(percentile95([1, 3, 2, 100, 4])).toBe(100)
    expect(percentile95([])).toBe(0)
    expect(framesPerSecond([10, 20, 20, 10])).toBeCloseTo(66.667, 2)
    expect(formatDiagnostic(Number.NaN)).toBe('unavailable')
  })

  it('creates explicit position and color buffers that can be disposed', () => {
    const geometry = createLineGeometry(
      new Float32Array([0, 0, 0, 1, 1, 1]),
      new Float32Array([1, 0, 0, 0, 1, 0]),
    )
    const dispose = vi.spyOn(geometry, 'dispose')

    expect(geometry.getAttribute('position').count).toBe(2)
    expect(geometry.getAttribute('color').count).toBe(2)
    expect(geometry.boundingSphere).not.toBeNull()
    geometry.dispose()
    expect(dispose).toHaveBeenCalledOnce()
  })
})
