import { Canvas } from '@react-three/fiber'
import './App.css'

function App() {
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
            <ambientLight intensity={0.7} />
            <pointLight position={[4, 5, 6]} intensity={30} color="#8cf5df" />
            <mesh>
              <icosahedronGeometry args={[1.25, 2]} />
              <meshStandardMaterial color="#123d4a" wireframe />
            </mesh>
          </Canvas>
          <div className="empty-state">
            <p className="empty-state__title">
              Open a repository to build its graph
            </p>
            <p>Source stays on this machine. No AI API or telemetry is used.</p>
          </div>
        </div>
      </section>
    </main>
  )
}

export default App
