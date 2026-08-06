import { Html } from '@react-three/drei'
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedMesh, Matrix4, Vector3 } from 'three'
import { buildAccessPoints } from '../activity/accessPoints'
import type { ActivityMode } from '../activity/extrusions'
import type { ActivityDisplaySettings } from '../activity/displaySettings'
import { CameraFocusController } from '../camera/CameraFocusController'
import { nodeColors } from '../graph/palette'
import type { GraphSnapshot, LiveFocusState, TrailAccess } from '../graph/types'
import { useLineGeometry } from '../rendering/lineGeometry'
import {
  nodeGeometryDetail,
  selectAccessPointsForRender,
  selectActivityAccesses,
  shouldShowHoverLabel,
} from '../rendering/lod'
import { buildStructurePositions } from '../rendering/structurePositions'
import { focusRoleForNode } from './focusState'
import { AccessPoints } from './AccessPoints'
import { RenderDiagnostics } from './RenderDiagnostics'
import { SessionTrail } from './SessionTrail'
import { TimeExtrusions } from './TimeExtrusions'
import { selectTrailAccesses, type TrailOptions } from './trail'
import { WorkExtrusions } from './WorkExtrusions'

const focusColors = {
  current: '#35ffd2',
  previous: '#ffb45f',
  secondary: '#d59aff',
  older: '#4d9aa0',
}
export function GraphScene({
  graph,
  selectedId,
  selectedAccessSequence,
  recenterKey,
  showLabels,
  showStructure,
  showActivity,
  showAccessPoints,
  showTrail,
  showDiagnostics,
  recentTrailAccesses,
  completeTrail,
  hideTrailRepeats,
  activityMode,
  activitySettings,
  focusState,
  cameraFocusId,
  onSelect,
  onInspectAccess,
  onManualInteraction,
}: {
  graph: GraphSnapshot
  selectedId?: string
  selectedAccessSequence?: number
  recenterKey: number
  showLabels: boolean
  showStructure: boolean
  showActivity: boolean
  showAccessPoints: boolean
  showTrail: boolean
  showDiagnostics: boolean
  recentTrailAccesses: number
  completeTrail: boolean
  hideTrailRepeats: boolean
  activityMode: ActivityMode
  activitySettings: ActivityDisplaySettings
  focusState?: LiveFocusState
  cameraFocusId?: string
  onSelect: (id: string) => void
  onInspectAccess: (access: TrailAccess) => void
  onManualInteraction: () => void
}) {
  const mesh = useRef<InstancedMesh>(null)
  const [hovered, setHovered] = useState<number>()
  const [nowMs, setNowMs] = useState(() => Date.now())
  const selected = graph.nodes.find((n) => n.id === selectedId)
  const cameraFocus = graph.nodes.find((n) => n.id === cameraFocusId)
  const trail = focusState?.trail ?? []
  const trailOptions = useMemo<TrailOptions>(
    () => ({
      recentAccesses: recentTrailAccesses,
      completeSession: completeTrail,
      hideConsecutiveRepeats: hideTrailRepeats,
    }),
    [completeTrail, hideTrailRepeats, recentTrailAccesses],
  )
  const olderTrailNodeIds = useMemo(() => {
    if (!showTrail) return new Set<string>()
    return new Set(
      selectTrailAccesses(focusState?.trail ?? [], trailOptions)
        .map((access) => access.node_id)
        .filter((id): id is string => Boolean(id)),
    )
  }, [focusState?.trail, showTrail, trailOptions])
  useLayoutEffect(() => {
    const matrix = new Matrix4()
    graph.nodes.forEach((node, index) => {
      matrix.makeTranslation(...node.position)
      const focusRole = focusRoleForNode(node.id, focusState, olderTrailNodeIds)
      const isCurrent = focusRole === 'current'
      const isPrevious = focusRole === 'previous'
      const isSecondary = focusRole === 'secondary'
      const isOlder = focusRole === 'older'
      const baseScale =
        node.kind === 'root' ? 1.7 : node.kind === 'directory' ? 1.3 : 1
      const scale =
        baseScale *
        (isCurrent
          ? 1.65
          : isPrevious
            ? 1.32
            : isSecondary
              ? 1.22
              : isOlder
                ? 1.08
                : 1)
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
                : isOlder
                  ? focusColors.older
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
  }, [focusState, graph, olderTrailNodeIds, selectedId])
  const positions = useMemo(() => {
    return buildStructurePositions(graph.nodes, graph.edges)
  }, [graph])
  const structureGeometry = useLineGeometry(positions)
  const activitySelection = useMemo(
    () => selectActivityAccesses(trail, selectedAccessSequence),
    [selectedAccessSequence, trail],
  )
  const pointSelection = useMemo(
    () =>
      selectAccessPointsForRender(
        buildAccessPoints(trail, graph.nodes),
        selectedAccessSequence,
      ),
    [graph.nodes, selectedAccessSequence, trail],
  )
  const nodeDetail = nodeGeometryDetail(graph.nodes.length)
  const hasActiveTimeInterval =
    activityMode === 'time' &&
    Boolean(focusState?.trail.some((access) => !access.ended_at))
  useEffect(() => {
    if (!hasActiveTimeInterval) return
    const timer = window.setInterval(() => setNowMs(Date.now()), 250)
    return () => window.clearInterval(timer)
  }, [hasActiveTimeInterval])
  const hoverNode = hovered === undefined ? undefined : graph.nodes[hovered]
  return (
    <>
      <ambientLight intensity={0.75} />
      <directionalLight position={[5, 8, 6]} intensity={2.5} color="#b9fff0" />
      {showStructure && (
        <lineSegments>
          <primitive object={structureGeometry} attach="geometry" />
          <lineBasicMaterial color="#294154" transparent opacity={0.72} />
        </lineSegments>
      )}
      {showActivity && activityMode === 'time' && (
        <TimeExtrusions
          nodes={graph.nodes}
          trail={activitySelection.items}
          nowMs={nowMs}
          scale={activitySettings.scale}
          visualCapMs={activitySettings.durationCapMs}
          onInspect={onInspectAccess}
        />
      )}
      {showActivity && activityMode === 'work' && (
        <WorkExtrusions
          nodes={graph.nodes}
          trail={activitySelection.items}
          scale={activitySettings.scale}
          visualCapLines={activitySettings.workCapLines}
          onInspect={onInspectAccess}
        />
      )}
      {showAccessPoints && (
        <AccessPoints
          points={pointSelection.items}
          onInspect={onInspectAccess}
        />
      )}
      {showTrail && (
        <SessionTrail
          nodes={graph.nodes}
          trail={focusState?.trail ?? []}
          options={trailOptions}
        />
      )}
      <instancedMesh
        ref={mesh}
        args={[undefined, undefined, graph.nodes.length]}
        onPointerMove={(e) => {
          e.stopPropagation()
          setHovered(
            shouldShowHoverLabel(graph.nodes.length, e.distance)
              ? e.instanceId
              : undefined,
          )
        }}
        onPointerOut={() => setHovered(undefined)}
        onClick={(e) => {
          e.stopPropagation()
          if (e.instanceId !== undefined) {
            onSelect(graph.nodes[e.instanceId]?.id ?? '')
          }
        }}
      >
        <icosahedronGeometry args={[0.22, nodeDetail]} />
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
      {((showActivity && activitySelection.active) ||
        (showAccessPoints && pointSelection.active)) && (
        <Html fullscreen zIndexRange={[30, 0]}>
          <p className="render-lod-status" role="status">
            Dense history LOD
            {showActivity && activitySelection.active
              ? ` · activity ${activitySelection.items.length} of ${activitySelection.totalCount}`
              : ''}
            {showAccessPoints && pointSelection.active
              ? ` · points ${pointSelection.items.length} of ${pointSelection.totalCount}`
              : ''}
            {' · '}active and inspected evidence retained
          </p>
        </Html>
      )}
      <CameraFocusController
        focus={cameraFocus?.position}
        recenterKey={recenterKey}
        onManualInteraction={onManualInteraction}
      />
      <RenderDiagnostics enabled={showDiagnostics} />
    </>
  )
}
