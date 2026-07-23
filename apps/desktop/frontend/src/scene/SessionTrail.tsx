import { Html } from '@react-three/drei'
import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4, Quaternion, Vector3 } from 'three'
import type { GraphNode, TrailAccess } from '../graph/types'
import {
  buildTrailSegments,
  formatTrailTimestamp,
  type TrailOptions,
} from './trail'

export function SessionTrail({
  nodes,
  trail,
  options,
}: {
  nodes: readonly GraphNode[]
  trail: readonly TrailAccess[]
  options: TrailOptions
}) {
  const arrows = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<number>()
  const segments = useMemo(
    () => buildTrailSegments(trail, nodes, options),
    [nodes, options, trail],
  )
  const positions = useMemo(
    () =>
      new Float32Array(
        segments.flatMap((segment) => [...segment.start, ...segment.end]),
      ),
    [segments],
  )
  const colors = useMemo(() => {
    const oldest = new Color('#52657a')
    const newest = new Color('#35ffd2')
    return new Float32Array(
      segments.flatMap((segment) => {
        const color = oldest.clone().lerp(newest, segment.recency)
        return [color.r, color.g, color.b, color.r, color.g, color.b]
      }),
    )
  }, [segments])

  useLayoutEffect(() => {
    const matrix = new Matrix4()
    const rotation = new Quaternion()
    const up = new Vector3(0, 1, 0)
    const scale = new Vector3(1, 1, 1)
    segments.forEach((segment, index) => {
      const start = new Vector3(...segment.start)
      const direction = new Vector3(...segment.end).sub(start).normalize()
      rotation.setFromUnitVectors(up, direction)
      matrix.compose(new Vector3(...segment.arrow), rotation, scale)
      arrows.current?.setMatrixAt(index, matrix)
      arrows.current?.setColorAt(
        index,
        new Color('#7d8fa3').lerp(new Color('#35ffd2'), segment.recency),
      )
    })
    if (arrows.current) {
      arrows.current.instanceMatrix.needsUpdate = true
      if (arrows.current.instanceColor)
        arrows.current.instanceColor.needsUpdate = true
    }
  }, [segments])

  const hoveredSegment = hovered === undefined ? undefined : segments[hovered]
  if (segments.length === 0) return null

  return (
    <>
      <lineSegments renderOrder={2}>
        <bufferGeometry>
          <bufferAttribute attach="attributes-position" args={[positions, 3]} />
          <bufferAttribute attach="attributes-color" args={[colors, 3]} />
        </bufferGeometry>
        <lineBasicMaterial
          vertexColors
          transparent
          opacity={0.94}
          depthTest={false}
        />
      </lineSegments>
      <instancedMesh
        ref={arrows}
        args={[undefined, undefined, segments.length]}
        renderOrder={3}
        onPointerMove={(event) => {
          event.stopPropagation()
          setHovered(event.instanceId)
        }}
        onPointerOut={() => setHovered(undefined)}
      >
        <coneGeometry args={[0.12, 0.36, 8]} />
        <meshBasicMaterial vertexColors depthTest={false} />
      </instancedMesh>
      {hoveredSegment && (
        <Html position={hoveredSegment.midpoint} center distanceFactor={9}>
          <span className="trail-tooltip">
            <strong>
              #{hoveredSegment.toSequence} {hoveredSegment.operation}
            </strong>
            <small>{formatTrailTimestamp(hoveredSegment.timestamp)}</small>
            <small>
              {hoveredSegment.source} · {hoveredSegment.confidence}
            </small>
          </span>
        </Html>
      )}
    </>
  )
}
