import { Html } from '@react-three/drei'
import { memo, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4, Vector3 } from 'three'
import { buildWorkGeometry } from '../activity/workExtrusions'
import type { DurationScale } from '../activity/timeExtrusions'
import type { GraphNode, TrailAccess, Vec3 } from '../graph/types'
import { formatWorkDelta } from '../inspector/format'
import { useLineGeometry } from '../rendering/lineGeometry'
import { markerGeometrySegments } from '../rendering/lod'

const additionColor = '#5ce0c4'
const deletionColor = '#ff7894'
const markerColors: Record<NonNullable<TrailAccess['work_status']>, string> = {
  known: '#5ce0c4',
  empty: '#8aa0ad',
  unknown: '#d59aff',
  binary: '#ffcf70',
  unsupported_encoding: '#ff9a70',
  pending: '#8bc6ff',
}

interface HoveredWork {
  access: TrailAccess
  position: Vec3
  clamped: boolean
}

export const WorkExtrusions = memo(function WorkExtrusions({
  nodes,
  trail,
  scale = 'log',
  visualCapLines,
  onInspect,
}: {
  nodes: readonly GraphNode[]
  trail: readonly TrailAccess[]
  scale?: DurationScale
  visualCapLines?: number
  onInspect: (access: TrailAccess) => void
}) {
  const endpoints = useRef<InstancedMesh>(null)
  const markers = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<HoveredWork>()
  const geometry = useMemo(
    () => buildWorkGeometry(trail, nodes, scale, visualCapLines),
    [nodes, scale, trail, visualCapLines],
  )
  const positions = useMemo(
    () =>
      new Float32Array(
        geometry.segments.flatMap((segment) => [
          ...segment.start,
          ...segment.end,
        ]),
      ),
    [geometry.segments],
  )
  const colors = useMemo(
    () =>
      new Float32Array(
        geometry.segments.flatMap((segment) => {
          const color = new Color(
            segment.direction === 'addition' ? additionColor : deletionColor,
          )
          return [color.r, color.g, color.b, color.r, color.g, color.b]
        }),
      ),
    [geometry.segments],
  )
  const lineGeometry = useLineGeometry(positions, colors)
  const sphereSegments = markerGeometrySegments(geometry.segments.length)

  useLayoutEffect(() => {
    const matrix = new Matrix4()
    geometry.segments.forEach((segment, index) => {
      matrix.makeTranslation(...segment.end)
      endpoints.current?.setMatrixAt(index, matrix)
      endpoints.current?.setColorAt(
        index,
        new Color(
          segment.direction === 'addition' ? additionColor : deletionColor,
        ),
      )
    })
    if (endpoints.current) {
      endpoints.current.instanceMatrix.needsUpdate = true
      if (endpoints.current.instanceColor)
        endpoints.current.instanceColor.needsUpdate = true
    }
  }, [geometry.segments])

  useLayoutEffect(() => {
    const matrix = new Matrix4()
    geometry.markers.forEach((marker, index) => {
      const size = marker.status === 'pending' ? 0.11 : 0.09
      matrix.makeTranslation(...marker.position)
      matrix.scale(new Vector3(size, size, size))
      markers.current?.setMatrixAt(index, matrix)
      markers.current?.setColorAt(index, new Color(markerColors[marker.status]))
    })
    if (markers.current) {
      markers.current.instanceMatrix.needsUpdate = true
      if (markers.current.instanceColor)
        markers.current.instanceColor.needsUpdate = true
    }
  }, [geometry.markers])

  if (geometry.segments.length === 0 && geometry.markers.length === 0)
    return null
  return (
    <>
      {geometry.segments.length > 0 && (
        <>
          <lineSegments renderOrder={3}>
            <primitive object={lineGeometry} attach="geometry" />
            <lineBasicMaterial vertexColors depthTest={false} />
          </lineSegments>
          <instancedMesh
            ref={endpoints}
            args={[undefined, undefined, geometry.segments.length]}
            renderOrder={4}
            onPointerMove={(event) => {
              event.stopPropagation()
              const segment =
                event.instanceId === undefined
                  ? undefined
                  : geometry.segments[event.instanceId]
              if (segment)
                setHovered({
                  access: segment.access,
                  position: segment.end,
                  clamped: segment.clamped,
                })
            }}
            onPointerOut={() => setHovered(undefined)}
            onClick={(event) => {
              event.stopPropagation()
              const segment =
                event.instanceId === undefined
                  ? undefined
                  : geometry.segments[event.instanceId]
              if (segment) onInspect(segment.access)
            }}
          >
            <sphereGeometry args={[0.11, sphereSegments, sphereSegments]} />
            <meshBasicMaterial vertexColors depthTest={false} />
          </instancedMesh>
        </>
      )}
      {geometry.markers.length > 0 && (
        <instancedMesh
          ref={markers}
          args={[undefined, undefined, geometry.markers.length]}
          renderOrder={4}
          onPointerMove={(event) => {
            event.stopPropagation()
            const marker =
              event.instanceId === undefined
                ? undefined
                : geometry.markers[event.instanceId]
            if (marker)
              setHovered({
                access: marker.access,
                position: marker.position,
                clamped: false,
              })
          }}
          onPointerOut={() => setHovered(undefined)}
          onClick={(event) => {
            event.stopPropagation()
            const marker =
              event.instanceId === undefined
                ? undefined
                : geometry.markers[event.instanceId]
            if (marker) onInspect(marker.access)
          }}
        >
          <octahedronGeometry args={[1, 0]} />
          <meshBasicMaterial vertexColors depthTest={false} />
        </instancedMesh>
      )}
      {hovered && (
        <Html position={hovered.position} center distanceFactor={9}>
          <span className="access-point-tooltip">
            <strong>Access #{hovered.access.sequence}</strong>
            <small>
              {hovered.access.work_status === 'binary'
                ? 'Binary file'
                : hovered.access.work_status === 'unsupported_encoding'
                  ? 'Unsupported encoding'
                  : hovered.access.work_status === 'pending'
                    ? 'Calculating work delta'
                    : formatWorkDelta(
                        hovered.access.lines_added,
                        hovered.access.lines_deleted,
                      )}
            </small>
            <small>
              {hovered.access.work_source ?? 'unknown'} ·{' '}
              {hovered.access.work_confidence ?? 'inferred'}
            </small>
            {hovered.clamped && (
              <small>Visual cap applied; exact values shown</small>
            )}
          </span>
        </Html>
      )}
    </>
  )
})
