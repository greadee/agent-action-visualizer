import type { Vec3 } from '../graph/types'

export function subtract(left: Vec3, right: Vec3): Vec3 {
  return [left[0] - right[0], left[1] - right[1], left[2] - right[2]]
}

export function normalize(vector: Vec3, fallback: Vec3 = [0, 0, 1]): Vec3 {
  const length = Math.hypot(vector[0], vector[1], vector[2])
  return length === 0
    ? fallback
    : [vector[0] / length, vector[1] / length, vector[2] / length]
}

export function addScaled(origin: Vec3, direction: Vec3, scale: number): Vec3 {
  return [
    origin[0] + direction[0] * scale,
    origin[1] + direction[1] * scale,
    origin[2] + direction[2] * scale,
  ]
}
