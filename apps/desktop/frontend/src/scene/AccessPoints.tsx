import { Html } from '@react-three/drei'
import { memo, useLayoutEffect, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4, Vector3 } from 'three'
import type { AccessPoint } from '../activity/accessPoints'
import type { TrailAccess } from '../graph/types'
import { markerGeometrySegments } from '../rendering/lod'
import { formatTrailTimestamp } from './trail'

export const AccessPoints = memo(function AccessPoints({
  points,
  onInspect,
}: {
  points: readonly AccessPoint[]
  onInspect: (access: TrailAccess) => void
}) {
  const mesh = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<number>()
  const sphereSegments = markerGeometrySegments(points.length)
  useLayoutEffect(() => {
    const matrix = new Matrix4()
    points.forEach((point, index) => {
      const size = point.aggregate
        ? 0.1 + Math.min(0.08, Math.log1p(point.accessCount) * 0.025)
        : point.active
          ? 0.11
          : 0.075
      matrix.makeTranslation(...point.position)
      matrix.scale(new Vector3(size, size, size))
      mesh.current?.setMatrixAt(index, matrix)
      mesh.current?.setColorAt(
        index,
        new Color(
          point.aggregate ? '#d59aff' : point.active ? '#35ffd2' : '#a8dbe2',
        ),
      )
    })
    if (mesh.current) {
      mesh.current.instanceMatrix.needsUpdate = true
      if (mesh.current.instanceColor)
        mesh.current.instanceColor.needsUpdate = true
    }
  }, [points])

  const hoveredPoint = hovered === undefined ? undefined : points[hovered]
  if (points.length === 0) return null
  const latest = hoveredPoint?.latest
  return (
    <>
      <instancedMesh
        ref={mesh}
        args={[undefined, undefined, points.length]}
        renderOrder={4}
        onPointerMove={(event) => {
          event.stopPropagation()
          setHovered(event.instanceId)
        }}
        onPointerOut={() => setHovered(undefined)}
        onClick={(event) => {
          event.stopPropagation()
          const point =
            event.instanceId === undefined
              ? undefined
              : points[event.instanceId]
          if (point) onInspect(point.latest)
        }}
      >
        <sphereGeometry args={[1, sphereSegments, sphereSegments]} />
        <meshBasicMaterial vertexColors depthTest={false} />
      </instancedMesh>
      {hoveredPoint && latest && (
        <Html position={hoveredPoint.position} center distanceFactor={9}>
          <span className="access-point-tooltip">
            <strong>
              {hoveredPoint.aggregate
                ? `${hoveredPoint.accessCount} accesses #${hoveredPoint.sequences[0]}–#${hoveredPoint.sequences.at(-1)}`
                : `Access #${latest.sequence}`}
            </strong>
            <small>{latest.operations.at(-1) ?? 'access'}</small>
            <small>{formatTrailTimestamp(latest.started_at)}</small>
            <small>
              {latest.source} · {latest.confidence}
            </small>
          </span>
        </Html>
      )}
    </>
  )
})
