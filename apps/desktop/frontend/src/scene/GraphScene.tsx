import { Html } from '@react-three/drei'
import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4, Vector3 } from 'three'
import { buildExtrusions, type ActivityMode } from '../activity/extrusions'
import { CameraFocusController } from '../camera/CameraFocusController'
import { nodeColors } from '../graph/palette'
import type { GraphSnapshot, LiveFocusState } from '../graph/types'
import { focusRoleForNode } from './focusState'

const focusColors = {
  current: '#35ffd2',
  previous: '#ffb45f',
  secondary: '#d59aff',
}
export function GraphScene({
  graph,
  selectedId,
  recenterKey,
  showLabels,
  showStructure,
  showActivity,
  activityMode,
  focusState,
  cameraFocusId,
  onSelect,
  onManualInteraction,
}: {
  graph: GraphSnapshot
  selectedId?: string
  recenterKey: number
  showLabels: boolean
  showStructure: boolean
  showActivity: boolean
  activityMode: ActivityMode
  focusState?: LiveFocusState
  cameraFocusId?: string
  onSelect: (id: string) => void
  onManualInteraction: () => void
}) {
  const mesh = useRef<InstancedMesh>(null)
  const activityPoints = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<number>()
  const selected = graph.nodes.find((n) => n.id === selectedId)
  const cameraFocus = graph.nodes.find((n) => n.id === cameraFocusId)
  useLayoutEffect(() => {
    const matrix = new Matrix4()
    graph.nodes.forEach((node, index) => {
      matrix.makeTranslation(...node.position)
      const focusRole = focusRoleForNode(node.id, focusState)
      const isCurrent = focusRole === 'current'
      const isPrevious = focusRole === 'previous'
      const isSecondary = focusRole === 'secondary'
      const baseScale =
        node.kind === 'root' ? 1.7 : node.kind === 'directory' ? 1.3 : 1
      const scale =
        baseScale *
        (isCurrent ? 1.65 : isPrevious ? 1.32 : isSecondary ? 1.22 : 1)
      matrix.scale(new Vector3(scale, scale, scale))
      mesh.current?.setMatrixAt(index, matrix)
      mesh.current?.setColorAt(
        index,
        new Color(
          isCurrent
            ? focusColors.current
            : isPrevious
              ? focusColors.previous
              : isSecondary
                ? focusColors.secondary
                : node.id === selectedId
                  ? '#ffffff'
                  : nodeColors[node.kind],
        ),
      )
    })
    if (mesh.current) {
      mesh.current.instanceMatrix.needsUpdate = true
      if (mesh.current.instanceColor)
        mesh.current.instanceColor.needsUpdate = true
    }
  }, [focusState, graph, selectedId])
  const positions = useMemo(() => {
    const byId = new Map(graph.nodes.map((n) => [n.id, n.position]))
    return new Float32Array(
      graph.edges.flatMap((e) => [
        ...(byId.get(e.source) ?? [0, 0, 0]),
        ...(byId.get(e.target) ?? [0, 0, 0]),
      ]),
    )
  }, [graph])
  const extrusions = useMemo(
    () => buildExtrusions(graph.nodes, activityMode),
    [activityMode, graph.nodes],
  )
  const activityPositions = useMemo(
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
      activityPoints.current?.setMatrixAt(index, matrix)
    })
    if (activityPoints.current) {
      activityPoints.current.instanceMatrix.needsUpdate = true
    }
  }, [extrusions, showActivity])
  const hoverNode = hovered === undefined ? undefined : graph.nodes[hovered]
  return (
    <>
      <ambientLight intensity={0.75} />
      <directionalLight position={[5, 8, 6]} intensity={2.5} color="#b9fff0" />
      {showStructure && (
        <lineSegments>
          <bufferGeometry>
            <bufferAttribute
              attach="attributes-position"
              args={[positions, 3]}
            />
          </bufferGeometry>
          <lineBasicMaterial color="#294154" transparent opacity={0.72} />
        </lineSegments>
      )}
      {showActivity && extrusions.length > 0 && (
        <>
          <lineSegments>
            <bufferGeometry>
              <bufferAttribute
                attach="attributes-position"
                args={[activityPositions, 3]}
              />
            </bufferGeometry>
            <lineBasicMaterial
              color={activityMode === 'time' ? '#ffb45f' : '#5ce0c4'}
            />
          </lineSegments>
          <instancedMesh
            ref={activityPoints}
            args={[undefined, undefined, extrusions.length]}
          >
            <sphereGeometry args={[0.12, 10, 10]} />
            <meshBasicMaterial
              color={activityMode === 'time' ? '#ffd096' : '#9affea'}
            />
          </instancedMesh>
        </>
      )}
      <instancedMesh
        ref={mesh}
        args={[undefined, undefined, graph.nodes.length]}
        onPointerMove={(e) => {
          e.stopPropagation()
          setHovered(e.instanceId)
        }}
        onPointerOut={() => setHovered(undefined)}
        onClick={(e) => {
          e.stopPropagation()
          if (e.instanceId !== undefined)
            onSelect(graph.nodes[e.instanceId]?.id ?? '')
        }}
      >
        <icosahedronGeometry args={[0.22, 2]} />
        <meshStandardMaterial roughness={0.35} metalness={0.15} />
      </instancedMesh>
      {showLabels && selected && (
        <Html position={selected.position} center distanceFactor={9}>
          <span className="node-label">{selected.path}</span>
        </Html>
      )}
      {showLabels && hoverNode && hoverNode.id !== selectedId && (
        <Html position={hoverNode.position} center distanceFactor={10}>
          <span className="node-label node-label--hover">{hoverNode.path}</span>
        </Html>
      )}
      <CameraFocusController
        focus={cameraFocus?.position}
        recenterKey={recenterKey}
        onManualInteraction={onManualInteraction}
      />
    </>
  )
}
