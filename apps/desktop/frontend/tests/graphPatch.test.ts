import { expect, it } from 'vitest'
import { applyGraphPatch } from '../src/graph/patch'
import type { GraphSnapshot } from '../src/graph/types'

it('applies ordered graph changes and ignores stale revisions', () => {
  const initial: GraphSnapshot = {
    revision: 1,
    nodes: [{ id: 'a', path: 'a.go', kind: 'source', position: [0, 0, 0] }],
    edges: [],
  }
  const next = applyGraphPatch(initial, {
    revision: 2,
    added: [{ id: 'b', path: 'b.go', kind: 'source', position: [1, 0, 0] }],
    updated: [
      { id: 'a', path: 'renamed.go', kind: 'source', position: [0, 0, 0] },
    ],
    removed: [],
    edges: [],
  })
  expect(next.nodes.map((node) => node.path)).toEqual(['b.go', 'renamed.go'])
  expect(
    applyGraphPatch(next, {
      revision: 1,
      added: [],
      updated: [],
      removed: [],
      edges: [],
    }),
  ).toBe(next)
})
