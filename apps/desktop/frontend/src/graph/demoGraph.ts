import type { GraphSnapshot } from './types'
export const demoGraph: GraphSnapshot = {
  nodes: [
    { id: 'root', path: '.', kind: 'root', position: [0, 0, 0] },
    { id: 'src', path: 'src', kind: 'directory', position: [-2.8, 1.4, 0.3] },
    {
      id: 'internal',
      path: 'internal',
      kind: 'directory',
      position: [2.4, 1.8, -0.8],
    },
    {
      id: 'app',
      path: 'src/App.tsx',
      kind: 'source',
      position: [-4.8, 2.8, 1.1],
    },
    {
      id: 'scene',
      path: 'src/scene/GraphScene.tsx',
      kind: 'source',
      position: [-4.2, -0.5, -1.6],
    },
    {
      id: 'session',
      path: 'internal/session/engine.go',
      kind: 'source',
      position: [4.7, 2.4, -1.2],
    },
    {
      id: 'tests',
      path: 'internal/session/engine_test.go',
      kind: 'test',
      position: [4.2, -0.8, 1.7],
    },
    {
      id: 'config',
      path: 'wails.json',
      kind: 'config',
      position: [0.5, -3.2, 0.8],
    },
  ],
  edges: [
    { source: 'root', target: 'src' },
    { source: 'root', target: 'internal' },
    { source: 'root', target: 'config' },
    { source: 'src', target: 'app' },
    { source: 'src', target: 'scene' },
    { source: 'internal', target: 'session' },
    { source: 'internal', target: 'tests' },
  ],
}
