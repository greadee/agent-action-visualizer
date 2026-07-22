import { Canvas } from '@react-three/fiber'
import { useState } from 'react'
import './App.css'
import { demoGraph } from './graph/demoGraph'
import { GraphScene } from './scene/GraphScene'

function App() {
  const [selectedId, setSelectedId] = useState('app')
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
          <p className="panel__value">No project selected</p>
          <button type="button">Open project</button>
          <div className="rule" />
          <p className="panel__label">MODE</p>
          <div className="segmented" aria-label="Activity mode">
            <button className="segmented__active" type="button">
              Time
            </button>
            <button type="button">Work</button>
          </div>
        </aside>
        <div className="viewport">
          <Canvas camera={{ position: [0, 0, 6], fov: 48 }}>
            <color attach="background" args={['#070a12']} />
            <GraphScene
              graph={demoGraph}
              selectedId={selectedId}
              onSelect={setSelectedId}
            />
          </Canvas>
          <div className="empty-state">
            <p className="empty-state__title">
              Deterministic project hierarchy
            </p>
            <p>Orbit, zoom, and select a node. Source stays on this machine.</p>
          </div>
        </div>
      </section>
    </main>
  )
}

export default App
