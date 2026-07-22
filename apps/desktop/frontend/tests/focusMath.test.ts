import { describe, expect, it } from 'vitest'
import { Vector3 } from 'three'
import {
  easeInOutCubic,
  focusCameraPosition,
  focusDurationMs,
} from '../src/camera/focusMath'

describe('camera focus math', () => {
  it('uses bounded distance-sensitive durations', () => {
    expect(focusDurationMs(0)).toBe(220)
    expect(focusDurationMs(4)).toBe(440)
    expect(focusDurationMs(100)).toBe(900)
  })

  it('clamps and eases progress symmetrically', () => {
    expect(easeInOutCubic(-1)).toBe(0)
    expect(easeInOutCubic(0.5)).toBe(0.5)
    expect(easeInOutCubic(2)).toBe(1)
  })

  it('places the camera radially outside the selected node', () => {
    const position = focusCameraPosition([3, 0, 0], [0, 0, 6], 5)
    expect(position.toArray()).toEqual([8, 0, 0])
    expect(position.distanceTo(new Vector3(3, 0, 0))).toBeCloseTo(5)
  })
})
