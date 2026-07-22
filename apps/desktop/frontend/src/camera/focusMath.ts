import { Vector3 } from 'three'
import type { Vec3 } from '../graph/types'

const MIN_DURATION_MS = 220
const MAX_DURATION_MS = 900

export function focusDurationMs(distance: number): number {
  return Math.round(
    Math.min(MAX_DURATION_MS, MIN_DURATION_MS + Math.max(0, distance) * 55),
  )
}

export function easeInOutCubic(progress: number): number {
  const value = Math.min(1, Math.max(0, progress))
  return value < 0.5
    ? 4 * value * value * value
    : 1 - Math.pow(-2 * value + 2, 3) / 2
}

export function focusCameraPosition(
  focus: Vec3,
  cameraPosition: Vec3,
  orbitDistance: number,
): Vector3 {
  const target = new Vector3(...focus)
  const outward = target.clone()
  if (outward.lengthSq() < 0.0001) {
    outward.set(...cameraPosition)
  }
  if (outward.lengthSq() < 0.0001) outward.set(0, 0, 1)
  return target.add(
    outward.normalize().multiplyScalar(Math.max(4, orbitDistance)),
  )
}
