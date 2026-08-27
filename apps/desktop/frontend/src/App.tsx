import { SegmentedControl } from '@prool-ui/react'
import {
  lazy,
  startTransition,
  Suspense,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import './App.css'
import type { ActivityMode } from './activity/extrusions'
import {
  buildSessionAnalytics,
  type AnalyticsBucket,
  type DirectoryActivity,
  type AgentActivity,
} from './analytics/sessionAnalytics'
import {
  DEFAULT_ACTIVITY_DISPLAY_SETTINGS,
  DURATION_VISUAL_CAPS_MS,
  WORK_VISUAL_CAPS_LINES,
  formatVisualCap,
} from './activity/displaySettings'
import { useGraphBridge } from './bridge/useGraphBridge'
import {
  DEFAULT_VISUALIZATION_FILTERS,
  applyVisualizationFilters,
  availableFilterValues,
  hasActiveFilters,
  normalizeVisualizationFilters,
  toggleFilterValue,
  type TimeRangeMode,
  type VisualizationFilters,
} from './filters/graphFilters'
import { demoGraph } from './graph/demoGraph'
import type { LiveFocusState, ReplaySession, TrailAccess } from './graph/types'
import { NodeInspector } from './inspector/NodeInspector'
import { formatDuration, formatWorkDelta } from './inspector/format'
import { scaleFixtureFromSearch } from './rendering/scaleFixture'
import { GraphLegend } from './scene/GraphLegend'
import { selectTrailAccesses } from './scene/trail'
import { adjacentAccessCursor, formatReplayTime } from './replay/timeline'

const reviewEpoch = Date.parse('2026-07-21T18:00:00Z')
const filterPreferenceKey = 'aav.visualizationFilters.v1'
const recentProjectPreferenceKey = 'aav.recentProject.v1'
const activityModePreferenceKey = 'aav.activityMode.v1'
const SceneViewport = lazy(() => import('./scene/SceneViewport'))

function App() {
  const scaleFixture = useMemo(
    () => scaleFixtureFromSearch(window.location.search, import.meta.env.DEV),
    [],
  )
  const [selectedId, setSelectedId] = useState('root')
  const [cameraFocusId, setCameraFocusId] = useState('root')
  const [recenterKey, setRecenterKey] = useState(0)
  const [query, setQuery] = useState('')
  const [showLabels, setShowLabels] = useState(true)
  const [showStructure, setShowStructure] = useState(true)
  const [showActivity, setShowActivity] = useState(true)
  const [showAccessPoints, setShowAccessPoints] = useState(true)
  const [showTrail, setShowTrail] = useState(true)
  const [showDiagnostics, setShowDiagnostics] = useState(
    scaleFixture?.diagnostics ?? false,
  )
  const [recentTrailAccesses, setRecentTrailAccesses] = useState(12)
  const [completeTrail, setCompleteTrail] = useState(false)
  const [hideTrailRepeats, setHideTrailRepeats] = useState(true)
  const [activityMode, setActivityMode] = useState<ActivityMode>(() =>
    loadActivityModePreference(),
  )
  const [activitySettings, setActivitySettings] = useState(
    DEFAULT_ACTIVITY_DISPLAY_SETTINGS,
  )
  const [projectPath, setProjectPath] = useState('')
  const [projectError, setProjectError] = useState('')
  const [loadedProjectPath, setLoadedProjectPath] = useState('')
  const [projectBusy, setProjectBusy] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const [health, setHealth] = useState<Record<string, string>>({
    status: 'checking',
    collector: 'checking',
    persistence: 'checking',
    recovery: 'none',
  })
  const [displayedFocus, setDisplayedFocus] = useState<
    LiveFocusState | undefined
  >(scaleFixture?.focus)
  const [selectedAccess, setSelectedAccess] = useState<TrailAccess>()
  const [autoFollow, setAutoFollow] = useState(true)
  const [livePaused, setLivePaused] = useState(false)
  const [manualHold, setManualHold] = useState(false)
  const [followDelaySeconds, setFollowDelaySeconds] = useState(5)
  const [persistedSessions, setPersistedSessions] = useState<
    ReplaySession['session'][]
  >([])
  const [replaySession, setReplaySession] = useState<ReplaySession>()
  const [isPlaying, setIsPlaying] = useState(false)
  const [playbackSpeed, setPlaybackSpeed] = useState(1)
  const [replayError, setReplayError] = useState('')
  const [filters, setFilters] = useState<VisualizationFilters>(() =>
    loadFilterPreferences(),
  )
  const reviewStep = useRef(0)
  const reviewStarted = useRef(false)
  const latestFocus = useRef<LiveFocusState | undefined>(undefined)
  const helpButton = useRef<HTMLButtonElement>(null)
  const helpCloseButton = useRef<HTMLButtonElement>(null)
  const {
    graph,
    liveFocus,
    getHealth,
    loadProject,
    pickProjectDirectory,
    publishActivityEvent,
    listPersistedSessions,
    replayPersistedSession,
  } = useGraphBridge(scaleFixture?.graph ?? demoGraph, scaleFixture?.focus)
  const isReplaying = Boolean(replaySession)
  const filterClockMs = replaySession?.cursor_at
    ? Date.parse(replaySession.cursor_at)
    : Date.now()
  const filtered = useMemo(
    () =>
      applyVisualizationFilters(
        graph,
        displayedFocus,
        filters,
        Number.isNaN(filterClockMs) ? Date.now() : filterClockMs,
      ),
    [displayedFocus, filterClockMs, filters, graph],
  )
  const filterOptions = useMemo(
    () => availableFilterValues(graph, displayedFocus),
    [displayedFocus, graph],
  )
  const filtersActive = hasActiveFilters(filters)
  const visibleGraph = filtered.graph
  const visibleFocus = filtered.focus
  const visibleFileCount = visibleGraph.nodes.filter(
    (node) => node.kind !== 'root' && node.kind !== 'directory',
  ).length
  const selected = visibleGraph.nodes.find((node) => node.id === selectedId)
  const matches = query
    ? visibleGraph.nodes
        .filter((node) => node.path.toLowerCase().includes(query.toLowerCase()))
        .slice(0, 5)
    : []
  const trailAccessCount = displayedFocus?.trail.length ?? 0
  const visibleActivityCount = visibleFocus?.trail.length ?? 0
  const visibleTrailAccessCount = selectTrailAccesses(
    visibleFocus?.trail ?? [],
    {
      recentAccesses: recentTrailAccesses,
      completeSession: completeTrail,
      hideConsecutiveRepeats: hideTrailRepeats,
    },
  ).length
  const analytics = useMemo(
    () => buildSessionAnalytics(visibleGraph, visibleFocus),
    [visibleGraph, visibleFocus],
  )
  const projectLoaded = Boolean(loadedProjectPath || scaleFixture)
  const loadedProjectName = projectName(loadedProjectPath)
  const collectorReady = health.collector === 'ready'
  const collectorChecking = health.collector === 'checking'

  useEffect(() => {
    if (!selected && visibleGraph.nodes[0])
      setSelectedId(visibleGraph.nodes[0].id)
  }, [selected, visibleGraph.nodes])

  useEffect(() => {
    saveFilterPreferences(filters)
  }, [filters])

  useEffect(() => {
    saveLocalPreference(activityModePreferenceKey, activityMode)
  }, [activityMode])

  useEffect(() => {
    let active = true
    const refreshHealth = () => {
      void getHealth()
        .then((next) => {
          if (active) setHealth(next)
        })
        .catch(() => {
          if (active)
            setHealth({
              status: 'degraded',
              collector: 'unavailable',
              persistence: 'unknown',
              recovery: 'unknown',
            })
        })
    }
    refreshHealth()
    const timer = window.setInterval(refreshHealth, 3000)
    return () => {
      active = false
      window.clearInterval(timer)
    }
  }, [])

  useEffect(() => {
    if (scaleFixture) return
    const recentProject = loadLocalPreference(recentProjectPreferenceKey)
    if (recentProject) void openLocalProject(recentProject, true)
  }, [])

  useEffect(() => {
    latestFocus.current = liveFocus
  }, [liveFocus])

  useEffect(() => {
    if (!selectedAccess || !visibleFocus) return
    const updated = visibleFocus.trail.find(
      (access) => access.sequence === selectedAccess.sequence,
    )
    if (updated && updated !== selectedAccess) setSelectedAccess(updated)
    if (!updated) setSelectedAccess(undefined)
  }, [selectedAccess, visibleFocus])

  useEffect(() => {
    if (livePaused || isReplaying || !liveFocus) return
    setDisplayedFocus(liveFocus)
    if (autoFollow && !manualHold && liveFocus.active_node_id) {
      setCameraFocusId(liveFocus.active_node_id)
      setSelectedId(liveFocus.active_node_id)
    }
  }, [autoFollow, isReplaying, liveFocus, livePaused, manualHold])

  useEffect(() => {
    if (!manualHold || !autoFollow || livePaused || isReplaying) return
    const timer = window.setTimeout(() => {
      setManualHold(false)
      if (latestFocus.current?.active_node_id) {
        setCameraFocusId(latestFocus.current.active_node_id)
        setSelectedId(latestFocus.current.active_node_id)
      }
    }, followDelaySeconds * 1000)
    return () => window.clearTimeout(timer)
  }, [autoFollow, followDelaySeconds, isReplaying, livePaused, manualHold])

  function inspectNode(id: string) {
    setSelectedId(id)
    setSelectedAccess(undefined)
    setCameraFocusId(id)
    if (autoFollow && !livePaused) setManualHold(true)
  }

  function updateFilters(next: Partial<VisualizationFilters>) {
    setFilters((current) =>
      normalizeVisualizationFilters({ ...current, ...next }),
    )
  }

  function toggleFilterList(
    key: 'fileTypes' | 'operations' | 'confidences' | 'agents',
    value: string,
  ) {
    setFilters((current) => ({
      ...current,
      [key]: toggleFilterValue(current[key], value),
    }))
  }

  function resetFilters() {
    setFilters(DEFAULT_VISUALIZATION_FILTERS)
    setQuery('')
  }

  function returnToLive() {
    setReplaySession(undefined)
    setIsPlaying(false)
    setReplayError('')
    setLivePaused(false)
    setManualHold(false)
    setDisplayedFocus(liveFocus)
    if (liveFocus?.active_node_id) {
      setSelectedId(liveFocus.active_node_id)
      setCameraFocusId(liveFocus.active_node_id)
    }
  }

  function closeHelp() {
    setHelpOpen(false)
    window.setTimeout(() => helpButton.current?.focus(), 0)
  }

  useEffect(() => {
    if (helpOpen) helpCloseButton.current?.focus()
  }, [helpOpen])

  async function openLocalProject(path: string, reopening = false) {
    setProjectBusy(true)
    setProjectError('')
    try {
      const resolved = await loadProject(path)
      setProjectPath(resolved)
      setLoadedProjectPath(resolved)
      saveLocalPreference(recentProjectPreferenceKey, resolved)
      setDisplayedFocus(undefined)
      setReplaySession(undefined)
      setPersistedSessions([])
      await refreshPersistedSessions()
    } catch (error) {
      if (reopening) removeLocalPreference(recentProjectPreferenceKey)
      const message = error instanceof Error ? error.message : String(error)
      setProjectError(
        reopening ? `Could not reopen the last project. ${message}` : message,
      )
    } finally {
      setProjectBusy(false)
    }
  }

  async function chooseLocalProject() {
    setProjectError('')
    try {
      const selectedPath = await pickProjectDirectory()
      if (selectedPath) await openLocalProject(selectedPath)
    } catch (error) {
      setProjectError(error instanceof Error ? error.message : String(error))
    }
  }

  async function refreshPersistedSessions() {
    try {
      setPersistedSessions(await listPersistedSessions())
    } catch (error) {
      setReplayError(error instanceof Error ? error.message : String(error))
    }
  }

  async function openReplay(sessionID: string, cursor: number) {
    try {
      const next = await replayPersistedSession(sessionID, cursor)
      startTransition(() => {
        setReplaySession(next)
        setDisplayedFocus(next.focus)
        setLivePaused(true)
        setManualHold(false)
        if (next.focus.active_node_id) {
          setSelectedId(next.focus.active_node_id)
          setCameraFocusId(next.focus.active_node_id)
        }
      })
      setReplayError('')
    } catch (error) {
      setReplayError(error instanceof Error ? error.message : String(error))
      setIsPlaying(false)
    }
  }

  useEffect(() => {
    if (!isPlaying || !replaySession) return
    const timer = window.setInterval(
      () => {
        const nextCursor = replaySession.cursor + 1
        if (nextCursor >= replaySession.event_count) {
          setIsPlaying(false)
          return
        }
        void openReplay(replaySession.session.id, nextCursor)
      },
      Math.max(120, 750 / playbackSpeed),
    )
    return () => window.clearInterval(timer)
  }, [isPlaying, playbackSpeed, replaySession])

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
    await addReviewBurst(60, 'p5-review-dense')
  }

  async function addScaleReviewBurst() {
    await addReviewBurst(1_000, 'p9-scale-burst')
  }

  async function addReviewBurst(count: number, eventPrefix: string) {
    const candidates = graph.nodes.filter(
      (node) => node.kind !== 'root' && node.kind !== 'directory',
    )
    if (candidates.length < 2) return
    await ensureReviewSession()
    const start = reviewStep.current
    for (let offset = 0; offset < count; offset += 1) {
      const active = candidates[offset % 2]!
      await publishActivityEvent({
        schema_version: '1.0',
        event_id: `${eventPrefix}-${start + offset}`,
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
    reviewStep.current += count
  }

  async function addDurationReviewBatch() {
    const candidates = graph.nodes.filter(
      (node) => node.kind !== 'root' && node.kind !== 'directory',
    )
    if (candidates.length < 3) return
    const now = Date.now()
    const sessionID = `p5-duration-review-${now}`
    const emit = async (
      eventID: string,
      nodeIndex: number | undefined,
      offsetMs: number,
      operation = 'read',
    ) =>
      publishActivityEvent({
        schema_version: '1.0',
        event_id: eventID,
        session_id: sessionID,
        source_type: 'synthetic',
        source_confidence: 'exact',
        event_type:
          nodeIndex === undefined
            ? 'session_started'
            : operation === 'patch'
              ? 'file_patched'
              : 'file_read',
        operation: nodeIndex === undefined ? undefined : operation,
        timestamp: new Date(now + offsetMs).toISOString(),
        path: nodeIndex === undefined ? undefined : candidates[nodeIndex]?.path,
      })
    await emit('p5-duration-start', undefined, -205_000)
    await emit('p5-duration-short', 0, -205_000)
    await emit('p5-duration-medium', 1, -204_000, 'patch')
    await emit('p5-duration-long', 2, -198_000)
    await emit('p5-duration-clamped', 0, -5_000, 'patch')
    await emit('p5-duration-active', 1, -1_000)
  }

  async function addWorkReviewBatch() {
    const candidates = graph.nodes.filter(
      (node) => node.kind !== 'root' && node.kind !== 'directory',
    )
    if (candidates.length < 3) return
    const now = Date.now()
    const sessionID = `p5-work-review-${now}`
    await publishActivityEvent({
      schema_version: '1.0',
      event_id: 'p5-work-start',
      session_id: sessionID,
      source_type: 'synthetic',
      source_confidence: 'exact',
      event_type: 'session_started',
      timestamp: new Date(now).toISOString(),
    })
    const review = [
      { added: 18, deleted: 0 },
      { added: 0, deleted: 12 },
      { added: 25, deleted: 9 },
      { added: 0, deleted: 0 },
      {},
      { binary: true },
      { added: 10_000, deleted: 5_000 },
    ]
    for (const [index, delta] of review.entries()) {
      await publishActivityEvent({
        schema_version: '1.0',
        event_id: `p5-work-${index}`,
        session_id: sessionID,
        source_type: 'synthetic',
        source_confidence: 'exact',
        event_type: 'file_patched',
        operation: 'patch',
        timestamp: new Date(now + index + 1).toISOString(),
        path: candidates[index % candidates.length]?.path,
        lines_added: delta.added,
        lines_deleted: delta.deleted,
        is_binary: delta.binary,
      })
    }
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <span className="eyebrow">LOCAL OBSERVABILITY</span>
          <h1>Agent Action Visualizer</h1>
        </div>
        <div className="topbar__actions">
          <button
            ref={helpButton}
            className="help-button"
            type="button"
            onClick={() => setHelpOpen(true)}
          >
            Setup &amp; Help
          </button>
          <div className="topbar__status">
            {scaleFixture && (
              <span className="scale-profile" aria-label="Scale fixture">
                SCALE {scaleFixture.nodeCount.toLocaleString()} NODES /{' '}
                {scaleFixture.accessCount.toLocaleString()} ACCESSES
              </span>
            )}
            <div
              className={`connection ${collectorReady ? '' : 'connection--disconnected'}`}
              aria-label="Collector status"
              aria-live="polite"
            >
              <span className="connection__dot" />{' '}
              {collectorChecking
                ? 'Checking local collector'
                : collectorReady
                  ? 'Live collector available'
                  : 'Collector disconnected · static graph available'}
            </div>
          </div>
        </div>
      </header>
      {helpOpen && (
        <div
          className="help-backdrop"
          role="presentation"
          onKeyDown={(event) => {
            if (event.key === 'Escape') closeHelp()
            if (event.key === 'Tab') {
              event.preventDefault()
              helpCloseButton.current?.focus()
            }
          }}
        >
          <section
            className="help-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="help-title"
          >
            <div className="help-dialog__header">
              <div>
                <span className="eyebrow">LOCAL SETUP</span>
                <h2 id="help-title">Setup &amp; troubleshooting</h2>
              </div>
              <button
                ref={helpCloseButton}
                type="button"
                aria-label="Close setup and help"
                onClick={closeHelp}
              >
                Close
              </button>
            </div>
            <div className="help-grid">
              <article>
                <h3>Open a project</h3>
                <p>
                  Choose a local Git repository. A deterministic static graph
                  appears immediately; live activity is optional.
                </p>
              </article>
              <article>
                <h3>Time and Work</h3>
                <p>
                  Every recorded access or edit keeps a stable point on its file
                  sphere. Time extends positive observed duration outward. Work
                  extends additions outward and deletions inward from that same
                  point. Turn Activity extrusions or Access points off to reduce
                  render work. Exact values remain in the inspector even when
                  geometry is capped.
                </p>
              </article>
              <article>
                <h3>Codex hook</h3>
                <p>
                  Keep <code>aav.exe</code> beside{' '}
                  <code>aav-codex-hook.exe</code>, then run the project-scope
                  install, status, and test commands. Uninstall removes only
                  AAV-managed hook entries.
                </p>
                <code className="help-command">
                  aav.exe codex install --scope project --project
                  &quot;&lt;repository&gt;&quot;
                </code>
                <code className="help-command">
                  aav.exe codex test --scope project --project
                  &quot;&lt;repository&gt;&quot;
                </code>
                <code className="help-command">
                  aav.exe codex uninstall --scope project --project
                  &quot;&lt;repository&gt;&quot;
                </code>
              </article>
              <article>
                <h3>Generic wrapper</h3>
                <p>
                  Place <code>aav-wrapper.exe</code> before a local agent
                  command. It preserves stdin, stdout, stderr, arguments, files,
                  and exit behavior if collection fails.
                </p>
                <code className="help-command">
                  aav-wrapper.exe --project-root &quot;&lt;repository&gt;&quot;
                  -- your-agent.exe
                </code>
              </article>
              <article>
                <h3>Local data</h3>
                <p>
                  Session metadata stays in{' '}
                  <code>%AppData%\agent-action-visualizer\sessions.db</code>.
                  Source contents, prompts, command arguments, and output are
                  not stored.
                </p>
              </article>
              <article>
                <h3>Common fixes</h3>
                <p>
                  Disconnected collector: keep using the static graph, then
                  restart the app. No history: run a configured hook or wrapper.
                  Invalid project: choose a readable folder containing{' '}
                  <code>.git</code>.
                </p>
              </article>
            </div>
          </section>
        </div>
      )}
      <section className="workspace" aria-label="Project graph workspace">
        <aside className="panel">
          {!projectLoaded && (
            <div className="first-run" role="status">
              <strong>Choose your first repository</strong>
              <p>
                Start with a private, local static graph. Add a Codex hook or
                generic wrapper later for live activity.
              </p>
            </div>
          )}
          <label className="panel__label" htmlFor="project-path">
            PROJECT
          </label>
          <input
            id="project-path"
            placeholder="Absolute project path"
            value={projectPath}
            aria-invalid={Boolean(projectError)}
            aria-describedby={projectError ? 'project-error' : undefined}
            onChange={(event) => setProjectPath(event.target.value)}
          />
          <div className="project-actions">
            <button
              type="button"
              disabled={projectBusy}
              onClick={() => void chooseLocalProject()}
            >
              Choose folder…
            </button>
            <button
              type="button"
              disabled={projectBusy}
              onClick={() => void openLocalProject(projectPath)}
            >
              {projectBusy ? 'Opening…' : 'Open path'}
            </button>
          </div>
          {loadedProjectName && (
            <p className="project-current" aria-live="polite">
              <span>Current repository</span>
              <strong>{loadedProjectName}</strong>
              <small>
                {displayedFocus?.trail.length
                  ? 'Live or replay history available'
                  : 'Static graph · no activity history yet'}
              </small>
            </p>
          )}
          {projectError && (
            <p id="project-error" className="error" role="alert">
              {projectError}
            </p>
          )}
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
          <p className="panel__label">FILTERS</p>
          <label className="panel__label" htmlFor="path-filter">
            PATH / DIRECTORY
          </label>
          <input
            id="path-filter"
            type="search"
            placeholder="Filter visible paths"
            value={filters.path}
            onChange={(event) => updateFilters({ path: event.target.value })}
          />
          <label className="panel__label" htmlFor="time-filter">
            TIME RANGE
          </label>
          <select
            id="time-filter"
            aria-label="Time range filter"
            value={filters.timeRange}
            onChange={(event) =>
              updateFilters({ timeRange: event.target.value as TimeRangeMode })
            }
          >
            <option value="all">All time</option>
            <option value="last-hour">Last hour</option>
            <option value="last-day">Last day</option>
            <option value="custom">Custom range</option>
          </select>
          {filters.timeRange === 'custom' && (
            <div className="filter-range">
              <input
                aria-label="Filter start time"
                type="datetime-local"
                value={filters.customStart}
                onChange={(event) =>
                  updateFilters({ customStart: event.target.value })
                }
              />
              <input
                aria-label="Filter end time"
                type="datetime-local"
                value={filters.customEnd}
                onChange={(event) =>
                  updateFilters({ customEnd: event.target.value })
                }
              />
            </div>
          )}
          <FilterGroup
            label="File type filters"
            options={filterOptions.fileTypes}
            selected={filters.fileTypes}
            onToggle={(value) => toggleFilterList('fileTypes', value)}
          />
          <FilterGroup
            label="Operation filters"
            options={filterOptions.operations}
            selected={filters.operations}
            onToggle={(value) => toggleFilterList('operations', value)}
          />
          <FilterGroup
            label="Confidence filters"
            options={filterOptions.confidences}
            selected={filters.confidences}
            onToggle={(value) => toggleFilterList('confidences', value)}
          />
          <FilterGroup
            label="Agent filters"
            options={filterOptions.agents}
            selected={filters.agents}
            onToggle={(value) => toggleFilterList('agents', value)}
          />
          <div className="filter-summary" aria-live="polite">
            <span>
              {visibleGraph.nodes.length} of {graph.nodes.length} nodes
            </span>
            <span>
              {filtered.visibleAccesses} of {trailAccessCount} accesses
            </span>
          </div>
          <button
            type="button"
            disabled={!filtersActive && !query}
            onClick={resetFilters}
          >
            Reset filters
          </button>
          {filtersActive && visibleFileCount === 0 && (
            <p className="replay-empty">No files match the current filters.</p>
          )}
          <div className="rule" />
          <p className="panel__label">SESSION ANALYTICS</p>
          <div className="analytics-panel" aria-live="polite">
            <div className="analytics-kpis">
              <MetricCard label="Accesses" value={analytics.totalAccesses} />
              <MetricCard label="Files" value={analytics.uniqueFiles} />
              <MetricCard
                label="Time"
                value={formatDuration(analytics.totalDurationMs)}
              />
              <MetricCard
                label="Work"
                value={formatWorkDelta(
                  analytics.linesAdded,
                  analytics.linesDeleted,
                )}
              />
            </div>
            <p className="analytics-note">
              {filtersActive
                ? 'Filtered session view'
                : isReplaying
                  ? 'Replay cursor view'
                  : 'Current session view'}
              {analytics.unknownDurationCount > 0
                ? `; ${analytics.unknownDurationCount} unknown durations`
                : ''}
              {analytics.unknownWorkCount +
                analytics.binaryWorkCount +
                analytics.unsupportedWorkCount +
                analytics.pendingWorkCount >
              0
                ? `; ${analytics.unknownWorkCount} unknown, ${analytics.binaryWorkCount} binary, ${analytics.unsupportedWorkCount} unsupported, ${analytics.pendingWorkCount} pending work`
                : ''}
            </p>
            <div className="analytics-ops">
              <span>create {analytics.operationCounts.created}</span>
              <span>modify {analytics.operationCounts.modified}</span>
              <span>delete {analytics.operationCounts.deleted}</span>
              <span>read {analytics.operationCounts.read}</span>
              <span>other {analytics.operationCounts.other}</span>
            </div>
            <AnalyticsList
              title="Top files"
              items={analytics.fileActivity.slice(0, 4)}
            />
            <DirectoryAnalyticsList
              title="Directories"
              items={analytics.directoryActivity.slice(0, 4)}
            />
            <AgentAnalyticsList
              title="Agents"
              items={analytics.agentActivity.slice(0, 4)}
            />
            <AnalyticsList
              title="Event confidence"
              items={analytics.confidenceCounts}
            />
            <AnalyticsList
              title="Work confidence"
              items={analytics.workConfidenceCounts}
            />
          </div>
          <div className="rule" />
          <p className="panel__label">LIVE FOCUS</p>
          <div className="focus-readout" aria-live="polite">
            <span
              className={`live-state live-state--${isReplaying ? 'replay' : livePaused ? 'paused' : manualHold ? 'manual' : 'live'}`}
            >
              {isReplaying
                ? 'REPLAY'
                : livePaused
                  ? 'PAUSED'
                  : manualHold
                    ? 'INSPECTING'
                    : 'LIVE'}
            </span>
            <strong>
              {visibleFocus?.active_path ??
                (filtersActive
                  ? 'Filtered from current view'
                  : 'Waiting for focus event')}
            </strong>
            <small>
              {visibleFocus?.operation
                ? `${visibleFocus.operation} · ${visibleFocus.confidence ?? 'unknown'}`
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
                livePaused || isReplaying ? returnToLive() : setLivePaused(true)
              }
            >
              {livePaused || isReplaying ? 'Return live' : 'Pause updates'}
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
                <button
                  type="button"
                  onClick={() => void addScaleReviewBurst()}
                >
                  Add 1,000-event burst
                </button>
                <button
                  type="button"
                  onClick={() => void addDurationReviewBatch()}
                >
                  Add duration review batch
                </button>
                <button type="button" onClick={() => void addWorkReviewBatch()}>
                  Add work review batch
                </button>
              </>
            )}
          </div>
          <div className="rule" />
          <p className="panel__label">SESSION REPLAY</p>
          <div className="replay-panel" aria-live="polite">
            <select
              aria-label="Persisted session"
              value={replaySession?.session.id ?? ''}
              onFocus={() => void refreshPersistedSessions()}
              onChange={(event) => {
                const sessionID = event.target.value
                setIsPlaying(false)
                if (!sessionID) returnToLive()
                else void openReplay(sessionID, Number.MAX_SAFE_INTEGER)
              }}
            >
              <option value="">Live session</option>
              {persistedSessions.map((session) => (
                <option key={session.id} value={session.id}>
                  {session.id} · {formatReplayTime(session.started_at)}
                </option>
              ))}
            </select>
            {persistedSessions.length === 0 && (
              <p className="replay-empty">
                {projectLoaded
                  ? 'No activity history yet. The static graph remains available.'
                  : 'Open a repository to see its local session history.'}
              </p>
            )}
            {replaySession && (
              <>
                <p className="replay-position">
                  Event {replaySession.cursor + 1} of{' '}
                  {replaySession.event_count} ·{' '}
                  {formatReplayTime(replaySession.cursor_at)}
                </p>
                <input
                  aria-label="Replay timeline"
                  type="range"
                  min="0"
                  max={Math.max(0, replaySession.event_count - 1)}
                  value={Math.max(0, replaySession.cursor)}
                  onChange={(event) =>
                    void openReplay(
                      replaySession.session.id,
                      Number(event.target.value),
                    )
                  }
                />
                <div className="replay-actions">
                  <button
                    type="button"
                    aria-label="Previous access"
                    disabled={
                      adjacentAccessCursor(
                        replaySession.timeline,
                        replaySession.cursor,
                        -1,
                      ) === replaySession.cursor
                    }
                    onClick={() =>
                      void openReplay(
                        replaySession.session.id,
                        adjacentAccessCursor(
                          replaySession.timeline,
                          replaySession.cursor,
                          -1,
                        ),
                      )
                    }
                  >
                    Previous access
                  </button>
                  <button
                    type="button"
                    aria-label={isPlaying ? 'Pause replay' : 'Play replay'}
                    disabled={replaySession.event_count < 2}
                    onClick={() => setIsPlaying((playing) => !playing)}
                  >
                    {isPlaying ? 'Pause' : 'Play'}
                  </button>
                  <button
                    type="button"
                    aria-label="Next access"
                    disabled={
                      adjacentAccessCursor(
                        replaySession.timeline,
                        replaySession.cursor,
                        1,
                      ) === replaySession.cursor
                    }
                    onClick={() =>
                      void openReplay(
                        replaySession.session.id,
                        adjacentAccessCursor(
                          replaySession.timeline,
                          replaySession.cursor,
                          1,
                        ),
                      )
                    }
                  >
                    Next access
                  </button>
                </div>
                <label className="replay-speed">
                  Speed
                  <select
                    aria-label="Replay speed"
                    value={playbackSpeed}
                    onChange={(event) =>
                      setPlaybackSpeed(Number(event.target.value))
                    }
                  >
                    <option value="0.5">0.5×</option>
                    <option value="1">1×</option>
                    <option value="2">2×</option>
                    <option value="4">4×</option>
                  </select>
                </label>
              </>
            )}
            {replayError && <p className="error">{replayError}</p>}
          </div>
          <div className="rule" />
          <p className="panel__label">MODE</p>
          <SegmentedControl
            className="segmented"
            label="Activity mode"
            options={[
              { value: 'time', label: 'Time' },
              { value: 'work', label: 'Work' },
            ]}
            value={activityMode}
            onChange={setActivityMode}
          />
          <label className="panel__label" htmlFor="activity-scale">
            VISUAL SCALE
          </label>
          <select
            id="activity-scale"
            aria-label="Activity visual scale"
            value={activitySettings.scale}
            onChange={(event) =>
              setActivitySettings((settings) => ({
                ...settings,
                scale: event.target.value === 'linear' ? 'linear' : 'log',
              }))
            }
          >
            <option value="log">Logarithmic</option>
            <option value="linear">Linear</option>
          </select>
          <label className="panel__label" htmlFor="activity-visual-cap">
            {activityMode === 'time'
              ? 'DURATION VISUAL CAP'
              : 'WORK VISUAL CAP'}
          </label>
          <select
            id="activity-visual-cap"
            aria-label={
              activityMode === 'time'
                ? 'Duration visual cap'
                : 'Work visual cap'
            }
            value={
              activityMode === 'time'
                ? activitySettings.durationCapMs
                : activitySettings.workCapLines
            }
            onChange={(event) => {
              const cap = Number(event.target.value)
              setActivitySettings((settings) =>
                activityMode === 'time'
                  ? { ...settings, durationCapMs: cap }
                  : { ...settings, workCapLines: cap },
              )
            }}
          >
            {(activityMode === 'time'
              ? DURATION_VISUAL_CAPS_MS
              : WORK_VISUAL_CAPS_LINES
            ).map((cap) => (
              <option key={cap} value={cap}>
                {formatVisualCap(activityMode, cap)}
              </option>
            ))}
          </select>
          <p className="activity-scale-note" aria-live="polite">
            Visual cap changes length only. Exact durations and line counts stay
            available in tooltips and the inspector.
          </p>
          <div
            className="activity-geometry"
            role="group"
            aria-labelledby="activity-geometry-label"
          >
            <p className="panel__label" id="activity-geometry-label">
              ACTIVITY GEOMETRY
            </p>
            <p className="activity-geometry__status" role="status">
              {visibleActivityCount === 0
                ? 'No recorded access or edit points yet. The static graph remains available; use Setup & Help to configure a hook or wrapper.'
                : `${visibleActivityCount.toLocaleString()} recorded access or edit ${visibleActivityCount === 1 ? 'point' : 'points'}. Time extends duration outward; Work renders additions outward and deletions inward.`}
            </p>
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
              aria-label="Render diagnostics"
              type="checkbox"
              checked={showDiagnostics}
              onChange={(event) => setShowDiagnostics(event.target.checked)}
            />
            Render diagnostics
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
          <GraphLegend
            activityMode={activityMode}
            activitySettings={activitySettings}
          />
        </aside>
        <div className="viewport">
          <Suspense
            fallback={
              <div className="scene-loading" role="status">
                Loading local renderer
              </div>
            }
          >
            <SceneViewport
              graph={visibleGraph}
              selectedId={selectedId}
              selectedAccessSequence={selectedAccess?.sequence}
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
              focusState={visibleFocus}
              cameraFocusId={cameraFocusId}
              onSelect={inspectNode}
              onInspectAccess={(access) => {
                setSelectedAccess(access)
                setSelectedId(access.node_id ?? '')
              }}
              onManualInteraction={() => {
                if (autoFollow && !livePaused) setManualHold(true)
              }}
            />
          </Suspense>
          <button
            className="recenter"
            type="button"
            onClick={() => {
              const active = displayedFocus?.active_node_id
              if (
                active &&
                visibleGraph.nodes.some((node) => node.id === active)
              ) {
                setCameraFocusId(active)
                setSelectedId(active)
                setManualHold(false)
              }
              setRecenterKey((value) => value + 1)
            }}
          >
            {visibleFocus?.active_node_id
              ? 'Recenter active file'
              : 'Recenter selection'}
          </button>
          <div className="empty-state">
            <p className="empty-state__title">
              {!projectLoaded
                ? 'Choose a local repository'
                : visibleFileCount === 0
                  ? 'No visible files in this repository'
                  : 'Deterministic project hierarchy'}
            </p>
            <p>
              {!projectLoaded
                ? 'Use Choose folder to begin. Nothing is uploaded.'
                : visibleFileCount === 0
                  ? 'Check repository contents and ignore rules, then reopen it.'
                  : displayedFocus?.trail.length
                    ? 'Orbit, zoom, and select a node. Source stays on this machine.'
                    : 'Static graph ready. Configure a hook or wrapper for live history.'}
            </p>
          </div>
        </div>
        <NodeInspector
          node={selected}
          access={selectedAccess}
          activityMode={activityMode}
          activitySettings={activitySettings}
        />
      </section>
    </main>
  )
}

export default App

function MetricCard({
  label,
  value,
}: {
  label: string
  value: string | number
}) {
  return (
    <div className="analytics-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function AnalyticsList({
  title,
  items,
}: {
  title: string
  items: AnalyticsBucket[]
}) {
  return (
    <div className="analytics-list">
      <p>{title}</p>
      {items.length === 0 && <span>No observed values</span>}
      {items.map((item) => (
        <span key={item.label} title={item.label}>
          <strong>{item.label}</strong>
          <em>{item.count}</em>
        </span>
      ))}
    </div>
  )
}

function DirectoryAnalyticsList({
  title,
  items,
}: {
  title: string
  items: DirectoryActivity[]
}) {
  return (
    <div className="analytics-list">
      <p>{title}</p>
      {items.length === 0 && <span>No observed values</span>}
      {items.map((item) => (
        <span key={item.label} title={item.label}>
          <strong>{item.label}</strong>
          <em>
            {item.count} / {formatWorkDelta(item.linesAdded, item.linesDeleted)}
          </em>
        </span>
      ))}
    </div>
  )
}

function AgentAnalyticsList({
  title,
  items,
}: {
  title: string
  items: AgentActivity[]
}) {
  return (
    <div className="analytics-list">
      <p>{title}</p>
      {items.length === 0 && <span>No observed values</span>}
      {items.map((item) => (
        <span key={item.label} title={item.label}>
          <strong>{item.label}</strong>
          <em>
            {item.count} / {formatWorkDelta(item.linesAdded, item.linesDeleted)}
          </em>
        </span>
      ))}
    </div>
  )
}

function FilterGroup({
  label,
  options,
  selected,
  onToggle,
}: {
  label: string
  options: string[]
  selected: string[]
  onToggle: (value: string) => void
}) {
  return (
    <fieldset className="filter-group">
      <legend>{label}</legend>
      {options.length === 0 && <p>No observed values</p>}
      {options.map((option) => (
        <label key={option} className="toggle">
          <input
            type="checkbox"
            checked={selected.includes(option)}
            onChange={() => onToggle(option)}
          />
          {option}
        </label>
      ))}
    </fieldset>
  )
}

function loadFilterPreferences() {
  try {
    const stored = window.localStorage.getItem(filterPreferenceKey)
    if (!stored) return DEFAULT_VISUALIZATION_FILTERS
    return normalizeVisualizationFilters(
      JSON.parse(stored) as VisualizationFilters,
    )
  } catch {
    return DEFAULT_VISUALIZATION_FILTERS
  }
}

function saveFilterPreferences(filters: VisualizationFilters) {
  try {
    window.localStorage.setItem(
      filterPreferenceKey,
      JSON.stringify(normalizeVisualizationFilters(filters)),
    )
  } catch {
    // Local preference persistence is optional and must not affect rendering.
  }
}

function loadActivityModePreference(): ActivityMode {
  return loadLocalPreference(activityModePreferenceKey) === 'work'
    ? 'work'
    : 'time'
}

function loadLocalPreference(key: string) {
  try {
    return window.localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

function saveLocalPreference(key: string, value: string) {
  try {
    window.localStorage.setItem(key, value)
  } catch {
    // Local preferences are optional and never affect collector behavior.
  }
}

function removeLocalPreference(key: string) {
  try {
    window.localStorage.removeItem(key)
  } catch {
    // Failure to clear an optional preference must not block first-run recovery.
  }
}

function projectName(path: string) {
  return path.split(/[\\/]/).filter(Boolean).at(-1) ?? ''
}
