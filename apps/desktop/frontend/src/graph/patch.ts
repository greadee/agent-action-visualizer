import type { GraphPatch, GraphSnapshot } from './types'

export function applyGraphPatch(
  snapshot: GraphSnapshot,
  patch: GraphPatch,
): GraphSnapshot {
  if (snapshot.revision !== undefined && patch.revision <= snapshot.revision)
    return snapshot
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]))
  for (const id of patch.removed) nodes.delete(id)
  for (const node of [...patch.added, ...patch.updated])
    nodes.set(node.id, node)
  return {
    revision: patch.revision,
    nodes: [...nodes.values()].sort((a, b) => a.path.localeCompare(b.path)),
    edges: patch.edges.filter(
      (edge) => nodes.has(edge.source) && nodes.has(edge.target),
    ),
  }
}
