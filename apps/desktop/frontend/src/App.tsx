import { Canvas } from '@react-three/fiber'
import { useEffect, useState } from 'react'
import './App.css'
import { useGraphBridge } from './bridge/useGraphBridge'
import { demoGraph } from './graph/demoGraph'
import { NodeInspector } from './inspector/NodeInspector'
import { GraphLegend } from './scene/GraphLegend'
import { GraphScene } from './scene/GraphScene'

function App() {
  const [selectedId, setSelectedId] = useState('app')
  const [recenterKey, setRecenterKey] = useState(0)
  const [query, setQuery] = useState('')
  const [showLabels, setShowLabels] = useState(true)
  const [showStructure, setShowStructure] = useState(true)
  const [projectPath, setProjectPath] = useState('')
  const [projectError, setProjectError] = useState('')
  const { graph, loadProject } = useGraphBridge(demoGraph)
  const selected = graph.nodes.find((node) => node.id === selectedId)
  const matches = query
    ? graph.nodes
        .filter((node) => node.path.toLowerCase().includes(query.toLowerCase()))
        .slice(0, 5)
    : []
  useEffect(() => {
    if (!selected && graph.nodes[0]) setSelectedId(graph.nodes[0].id)
  }, [graph.nodes, selected])
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
                    setSelectedId(node.id)
                    setQuery('')
                  }}
                >
                  {node.path}
                </button>
              ))}
            </div>
          )}
          <div className="rule" />
          <p className="panel__label">MODE</p>
          <div className="segmented" aria-label="Activity mode">
            <button className="segmented__active" type="button">
              Time
            </button>
            <button type="button">Work</button>
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
          <GraphLegend />
        </aside>
        <div className="viewport">
          <Canvas camera={{ position: [0, 0, 6], fov: 48 }}>
            <color attach="background" args={['#070a12']} />
            <GraphScene
              graph={graph}
              selectedId={selectedId}
              recenterKey={recenterKey}
              showLabels={showLabels}
              showStructure={showStructure}
              onSelect={setSelectedId}
            />
          </Canvas>
          <button
            className="recenter"
            type="button"
            onClick={() => setRecenterKey((value) => value + 1)}
          >
            Recenter selection
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
