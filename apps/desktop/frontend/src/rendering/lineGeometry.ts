import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry } from 'three'

export function createLineGeometry(
  positions: Float32Array,
  colors?: Float32Array,
): BufferGeometry {
  const geometry = new BufferGeometry()
  geometry.setAttribute('position', new BufferAttribute(positions, 3))
  if (colors) geometry.setAttribute('color', new BufferAttribute(colors, 3))
  geometry.computeBoundingSphere()
  return geometry
}

export function useLineGeometry(
  positions: Float32Array,
  colors?: Float32Array,
): BufferGeometry {
  const geometry = useMemo(
    () => createLineGeometry(positions, colors),
    [colors, positions],
  )
  useEffect(() => () => geometry.dispose(), [geometry])
  return geometry
}
