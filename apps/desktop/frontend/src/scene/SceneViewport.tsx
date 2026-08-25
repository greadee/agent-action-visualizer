import { Canvas } from '@react-three/fiber'
import type { ActivityDisplaySettings } from '../activity/displaySettings'
import type { ActivityMode } from '../activity/extrusions'
import type { GraphSnapshot, LiveFocusState, TrailAccess } from '../graph/types'
import { GraphScene } from './GraphScene'

export default function SceneViewport({
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
  return (
    <Canvas
      camera={{ position: [0, 0, 28], fov: 48 }}
      dpr={[1, 1.5]}
      gl={{ antialias: true, powerPreference: 'high-performance' }}
    >
      <color attach="background" args={['#070a12']} />
      <GraphScene
        graph={graph}
        selectedId={selectedId}
        selectedAccessSequence={selectedAccessSequence}
        recenterKey={recenterKey}
        showLabels={showLabels}
        showStructure={showStructure}
        showActivity={showActivity}
        showAccessPoints={showAccessPoints}
        showTrail={showTrail}
        showDiagnostics={showDiagnostics}
        recentTrailAccesses={recentTrailAccesses}
        completeTrail={completeTrail}
        hideTrailRepeats={hideTrailRepeats}
        activityMode={activityMode}
        activitySettings={activitySettings}
        focusState={focusState}
        cameraFocusId={cameraFocusId}
        onSelect={onSelect}
        onInspectAccess={onInspectAccess}
        onManualInteraction={onManualInteraction}
      />
    </Canvas>
  )
}
