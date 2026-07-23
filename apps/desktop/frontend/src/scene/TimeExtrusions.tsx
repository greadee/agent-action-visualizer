import { Html } from '@react-three/drei'
import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4 } from 'three'
import {
  buildTimeExtrusions,
  type DurationScale,
} from '../activity/timeExtrusions'
import type { GraphNode, TrailAccess } from '../graph/types'
import { formatExactDuration } from '../inspector/format'

export function TimeExtrusions({
  nodes,
  trail,
  nowMs,
  scale = 'log',
  onInspect,
}: {
  nodes: readonly GraphNode[]
  trail: readonly TrailAccess[]
  nowMs: number
  scale?: DurationScale
  onInspect: (access: TrailAccess) => void
}) {
  const endpoints = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<number>()
  const extrusions = useMemo(
    () => buildTimeExtrusions(trail, nodes, nowMs, scale),
    [nodes, nowMs, scale, trail],
  )
  const positions = useMemo(
    () =>
      new Float32Array(
        extrusions.flatMap((extrusion) => [
          ...extrusion.start,
          ...extrusion.end,
        ]),
      ),
    [extrusions],
  )

  useLayoutEffect(() => {
    const matrix = new Matrix4()
    extrusions.forEach((extrusion, index) => {
      matrix.makeTranslation(...extrusion.end)
      endpoints.current?.setMatrixAt(index, matrix)
      endpoints.current?.setColorAt(
        index,
        new Color(extrusion.active ? '#fff3c9' : '#ffd096'),
      )
    })
    if (endpoints.current) {
      endpoints.current.instanceMatrix.needsUpdate = true
      if (endpoints.current.instanceColor)
        endpoints.current.instanceColor.needsUpdate = true
    }
  }, [extrusions])

  const selected = hovered === undefined ? undefined : extrusions[hovered]
  if (extrusions.length === 0) return null
  return (
    <>
      <lineSegments renderOrder={3}>
        <bufferGeometry>
          <bufferAttribute attach="attributes-position" args={[positions, 3]} />
        </bufferGeometry>
        <lineBasicMaterial color="#ffb45f" depthTest={false} />
      </lineSegments>
      <instancedMesh
        ref={endpoints}
        args={[undefined, undefined, extrusions.length]}
        renderOrder={4}
        onPointerMove={(event) => {
          event.stopPropagation()
          setHovered(event.instanceId)
        }}
        onPointerOut={() => setHovered(undefined)}
        onClick={(event) => {
          event.stopPropagation()
          const extrusion =
            event.instanceId === undefined
              ? undefined
              : extrusions[event.instanceId]
          if (extrusion) onInspect(extrusion.access)
        }}
      >
        <sphereGeometry args={[0.11, 10, 10]} />
        <meshBasicMaterial vertexColors depthTest={false} />
      </instancedMesh>
      {selected && (
        <Html position={selected.end} center distanceFactor={9}>
          <span className="access-point-tooltip">
            <strong>Access #{selected.sequence}</strong>
            <small>{formatExactDuration(selected.durationMs)}</small>
            <small>
              {selected.active ? 'Active interval' : 'Completed interval'}
            </small>
            <small>
              {selected.access.source} · {selected.access.confidence}
            </small>
          </span>
        </Html>
      )}
    </>
  )
}
