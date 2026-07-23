import { Canvas } from '@react-three/fiber'
import { useEffect, useRef, useState } from 'react'
import './App.css'
import type { ActivityMode } from './activity/extrusions'
import { useGraphBridge } from './bridge/useGraphBridge'
import { demoGraph } from './graph/demoGraph'
import type { LiveFocusState } from './graph/types'
import { NodeInspector } from './inspector/NodeInspector'
import { GraphLegend } from './scene/GraphLegend'
import { GraphScene } from './scene/GraphScene'
import { selectTrailAccesses } from './scene/trail'

const reviewEpoch = Date.parse('2026-07-21T18:00:00Z')

function App() {
  const [selectedId, setSelectedId] = useState('root')
  const [cameraFocusId, setCameraFocusId] = useState('root')
  const [recenterKey, setRecenterKey] = useState(0)
  const [query, setQuery] = useState('')
  const [showLabels, setShowLabels] = useState(true)
  const [showStructure, setShowStructure] = useState(true)
  const [showActivity, setShowActivity] = useState(true)
  const [showAccessPoints, setShowAccessPoints] = useState(true)
  const [showTrail, setShowTrail] = useState(true)
  const [recentTrailAccesses, setRecentTrailAccesses] = useState(12)
  const [completeTrail, setCompleteTrail] = useState(false)
  const [hideTrailRepeats, setHideTrailRepeats] = useState(true)
  const [activityMode, setActivityMode] = useState<ActivityMode>('time')
  const [projectPath, setProjectPath] = useState('')
  const [projectError, setProjectError] = useState('')
  const [displayedFocus, setDisplayedFocus] = useState<LiveFocusState>()
  const [autoFollow, setAutoFollow] = useState(true)
  const [livePaused, setLivePaused] = useState(false)
  const [manualHold, setManualHold] = useState(false)
  const [followDelaySeconds, setFollowDelaySeconds] = useState(5)
  const reviewStep = useRef(0)
  const reviewStarted = useRef(false)
  const latestFocus = useRef<LiveFocusState | undefined>(undefined)
  const { graph, liveFocus, loadProject, publishActivityEvent } =
    useGraphBridge(demoGraph)
  const selected = graph.nodes.find((node) => node.id === selectedId)
  const matches = query
    ? graph.nodes
        .filter((node) => node.path.toLowerCase().includes(query.toLowerCase()))
        .slice(0, 5)
    : []
  const trailAccessCount = displayedFocus?.trail.length ?? 0
  const visibleTrailAccessCount = selectTrailAccesses(
    displayedFocus?.trail ?? [],
    {
      recentAccesses: recentTrailAccesses,
      completeSession: completeTrail,
      hideConsecutiveRepeats: hideTrailRepeats,
    },
  ).length

  useEffect(() => {
    if (!selected && graph.nodes[0]) setSelectedId(graph.nodes[0].id)
  }, [graph.nodes, selected])

  useEffect(() => {
    latestFocus.current = liveFocus
  }, [liveFocus])

  useEffect(() => {
    if (livePaused || !liveFocus) return
    setDisplayedFocus(liveFocus)
    if (autoFollow && !manualHold && liveFocus.active_node_id) {
      setCameraFocusId(liveFocus.active_node_id)
      setSelectedId(liveFocus.active_node_id)
    }
  }, [autoFollow, liveFocus, livePaused, manualHold])

  useEffect(() => {
    if (!manualHold || !autoFollow || livePaused) return
    const timer = window.setTimeout(() => {
      setManualHold(false)
      if (latestFocus.current?.active_node_id) {
        setCameraFocusId(latestFocus.current.active_node_id)
        setSelectedId(latestFocus.current.active_node_id)
      }
    }, followDelaySeconds * 1000)
    return () => window.clearTimeout(timer)
  }, [autoFollow, followDelaySeconds, livePaused, manualHold])

  function inspectNode(id: string) {
    setSelectedId(id)
    setCameraFocusId(id)
    if (autoFollow && !livePaused) setManualHold(true)
  }

  function returnToLive() {
    setLivePaused(false)
    setManualHold(false)
    setDisplayedFocus(liveFocus)
    if (liveFocus?.active_node_id) {
      setSelectedId(liveFocus.active_node_id)
      setCameraFocusId(liveFocus.active_node_id)
    }
  }

  async function ensureReviewSession() {
    if (reviewStarted.current) return
    await publishActivityEvent({
      schema_version: '1.0',
      event_id: 'p4-review-start',
      session_id: 'p4-review',
      source_type: 'synthetic',
      source_confidence: 'exact',
      event_type: 'session_started',
      timestamp: new Date(reviewEpoch).toISOString(),
    })
    reviewStarted.current = true
  }

  async function advanceReviewFocus() {
    const candidates = graph.nodes.filter(
      (node) => node.kind !== 'root' && node.kind !== 'directory',
    )
    if (candidates.length === 0) return
    const step = reviewStep.current
    await ensureReviewSession()
    const active = candidates[step % candidates.length]!
    const secondary = [
      candidates[(step + 1) % candidates.length]!,
      candidates[(step + 2) % candidates.length]!,
    ].filter((node) => node.id !== active.id)
    await publishActivityEvent({
      schema_version: '1.0',
      event_id: `p4-review-${step}`,
      session_id: 'p4-review',
      source_type: 'synthetic',
      source_confidence: 'exact',
      event_type: step % 2 === 0 ? 'file_patched' : 'file_read',
      operation: step % 2 === 0 ? 'patch' : 'read',
      timestamp: new Date(reviewEpoch + (step + 1) * 1000).toISOString(),
      path: active.path,
      metadata: { secondary_paths: secondary.map((node) => node.path) },
    })
    reviewStep.current += 1
  }

  async function addDenseReviewBatch() {
    const candidates = graph.nodes.filter(
      (node) => node.kind !== 'root' && node.kind !== 'directory',
    )
    if (candidates.length < 2) return
    await ensureReviewSession()
    const start = reviewStep.current
    for (let offset = 0; offset < 60; offset += 1) {
      const active = candidates[offset % 2]!
      await publishActivityEvent({
        schema_version: '1.0',
        event_id: `p5-review-dense-${start + offset}`,
        session_id: 'p4-review',
        source_type: 'synthetic',
        source_confidence: 'exact',
        event_type: offset % 2 === 0 ? 'file_patched' : 'file_read',
        operation: offset % 2 === 0 ? 'patch' : 'read',
        timestamp: new Date(
          reviewEpoch + (start + offset + 1) * 1000,
        ).toISOString(),
        path: active.path,
      })
    }
    reviewStep.current += 60
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <span className="eyebrow">LOCAL OBSERVABILITY</span>
          <h1>Agent Action Visualizer</h1>
        </div>
        <div className="connection" aria-label="Collector status">
          <span className="connection__dot" /> Collector ready
        </div>
      </header>
      <section className="workspace" aria-label="Project graph workspace">
        <aside className="panel">
          <p className="panel__label">PROJECT</p>
          <input
            aria-label="Project path"
            placeholder="Absolute project path"
            value={projectPath}
            onChange={(event) => setProjectPath(event.target.value)}
          />
          <button
            type="button"
            onClick={() => {
              setProjectError('')
              void loadProject(projectPath).catch((error: unknown) =>
                setProjectError(
                  error instanceof Error ? error.message : String(error),
                ),
              )
            }}
          >
            Load project
          </button>
          {projectError && <p className="error">{projectError}</p>}
          <div className="rule" />
          <label className="panel__label" htmlFor="node-search">
            SEARCH
          </label>
          <input
            id="node-search"
            type="search"
            placeholder="Find a path"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          {matches.length > 0 && (
            <div className="search-results">
              {matches.map((node) => (
                <button
                  key={node.id}
                  type="button"
                  onClick={() => {
                    inspectNode(node.id)
                    setQuery('')
                  }}
                >
                  {node.path}
                </button>
              ))}
            </div>
          )}
          <div className="rule" />
          <p className="panel__label">LIVE FOCUS</p>
          <div className="focus-readout" aria-live="polite">
            <span
              className={`live-state live-state--${livePaused ? 'paused' : manualHold ? 'manual' : 'live'}`}
            >
              {livePaused ? 'PAUSED' : manualHold ? 'INSPECTING' : 'LIVE'}
            </span>
            <strong>
              {displayedFocus?.active_path ?? 'Waiting for focus event'}
            </strong>
            <small>
              {displayedFocus?.operation
                ? `${displayedFocus.operation} · ${displayedFocus.confidence ?? 'unknown'}`
                : 'No current operation'}
            </small>
          </div>
          <label className="toggle">
            <input
              aria-label="Auto-follow"
              type="checkbox"
              checked={autoFollow}
              onChange={(event) => {
                setAutoFollow(event.target.checked)
                setManualHold(false)
                if (event.target.checked && !livePaused) returnToLive()
              }}
            />
            Auto-follow active file
          </label>
          <label className="follow-delay">
            Resume after
            <input
              aria-label="Auto-follow delay seconds"
              type="number"
              min="1"
              max="60"
              value={followDelaySeconds}
              onChange={(event) =>
                setFollowDelaySeconds(
                  Math.min(60, Math.max(1, Number(event.target.value) || 1)),
                )
              }
            />
            sec
          </label>
          <div className="live-actions">
            <button
              type="button"
              onClick={() =>
                livePaused ? returnToLive() : setLivePaused(true)
              }
            >
              {livePaused ? 'Return live' : 'Pause updates'}
            </button>
            {import.meta.env.DEV && (
              <>
                <button type="button" onClick={() => void advanceReviewFocus()}>
                  Next review event
                </button>
                <button
                  type="button"
                  onClick={() => void addDenseReviewBatch()}
                >
                  Add dense review batch
                </button>
              </>
            )}
          </div>
          <div className="rule" />
          <p className="panel__label">MODE</p>
          <div className="segmented" aria-label="Activity mode">
            <button
              className={activityMode === 'time' ? 'segmented__active' : ''}
              type="button"
              onClick={() => setActivityMode('time')}
            >
              Time
            </button>
            <button
              className={activityMode === 'work' ? 'segmented__active' : ''}
              type="button"
              onClick={() => setActivityMode('work')}
            >
              Work
            </button>
          </div>
          <div className="rule" />
          <label className="panel__label" htmlFor="layout-select">
            LAYOUT
          </label>
          <select id="layout-select" defaultValue="spherical">
            <option value="spherical">Spherical hierarchy</option>
          </select>
          <label className="toggle">
            <input
              type="checkbox"
              checked={showStructure}
              onChange={(event) => setShowStructure(event.target.checked)}
            />
            Structure edges
          </label>
          <label className="toggle">
            <input
              type="checkbox"
              checked={showLabels}
              onChange={(event) => setShowLabels(event.target.checked)}
            />
            Labels
          </label>
          <label className="toggle">
            <input
              type="checkbox"
              checked={showActivity}
              onChange={(event) => setShowActivity(event.target.checked)}
            />
            Activity extrusions
          </label>
          <label className="toggle">
            <input
              aria-label="Access points"
              type="checkbox"
              checked={showAccessPoints}
              onChange={(event) => setShowAccessPoints(event.target.checked)}
            />
            Access points
          </label>
          <div className="rule" />
          <p className="panel__label">SESSION TRAIL</p>
          <label className="toggle">
            <input
              aria-label="Session trail"
              type="checkbox"
              checked={showTrail}
              onChange={(event) => setShowTrail(event.target.checked)}
            />
            Directional travel edges
          </label>
          {showTrail && (
            <div className="trail-controls">
              <p aria-live="polite">
                {visibleTrailAccessCount} of {trailAccessCount} accesses
              </p>
              <label className="toggle">
                <input
                  aria-label="Complete session trail"
                  type="checkbox"
                  checked={completeTrail}
                  onChange={(event) => setCompleteTrail(event.target.checked)}
                />
                Complete session
              </label>
              {!completeTrail && (
                <label className="trail-limit">
                  Last
                  <input
                    aria-label="Recent trail access limit"
                    type="number"
                    min="2"
                    max="200"
                    value={recentTrailAccesses}
                    onChange={(event) =>
                      setRecentTrailAccesses(
                        Math.min(
                          200,
                          Math.max(2, Number(event.target.value) || 2),
                        ),
                      )
                    }
                  />
                  accesses
                </label>
              )}
              <label className="toggle">
                <input
                  aria-label="Hide consecutive repeated accesses"
                  type="checkbox"
                  checked={hideTrailRepeats}
                  onChange={(event) =>
                    setHideTrailRepeats(event.target.checked)
                  }
                />
                Hide consecutive repeats
              </label>
            </div>
          )}
          <GraphLegend activityMode={activityMode} />
        </aside>
        <div className="viewport">
          <Canvas camera={{ position: [0, 0, 28], fov: 48 }}>
            <color attach="background" args={['#070a12']} />
            <GraphScene
              graph={graph}
              selectedId={selectedId}
              recenterKey={recenterKey}
              showLabels={showLabels}
              showStructure={showStructure}
              showActivity={showActivity}
              showAccessPoints={showAccessPoints}
              showTrail={showTrail}
              recentTrailAccesses={recentTrailAccesses}
              completeTrail={completeTrail}
              hideTrailRepeats={hideTrailRepeats}
              activityMode={activityMode}
              focusState={displayedFocus}
              cameraFocusId={cameraFocusId}
              onSelect={inspectNode}
              onManualInteraction={() => {
                if (autoFollow && !livePaused) setManualHold(true)
              }}
            />
          </Canvas>
          <button
            className="recenter"
            type="button"
            onClick={() => {
              const active = displayedFocus?.active_node_id
              if (active) {
                setCameraFocusId(active)
                setSelectedId(active)
                setManualHold(false)
              }
              setRecenterKey((value) => value + 1)
            }}
          >
            {displayedFocus?.active_node_id
              ? 'Recenter active file'
              : 'Recenter selection'}
          </button>
          <div className="empty-state">
            <p className="empty-state__title">
              Deterministic project hierarchy
            </p>
            <p>Orbit, zoom, and select a node. Source stays on this machine.</p>
          </div>
        </div>
        <NodeInspector node={selected} />
      </section>
    </main>
  )
}

export default App
