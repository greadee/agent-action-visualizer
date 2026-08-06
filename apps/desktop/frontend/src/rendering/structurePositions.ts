import type { GraphEdge, GraphNode } from '../graph/types'

export function buildStructurePositions(
  nodes: readonly GraphNode[],
  edges: readonly GraphEdge[],
): Float32Array {
  const positions = new Float32Array(edges.length * 6)
  const byId = new Map(nodes.map((node) => [node.id, node.position]))
  edges.forEach((edge, edgeIndex) => {
    const source = byId.get(edge.source)
    const target = byId.get(edge.target)
    const offset = edgeIndex * 6
    if (source) positions.set(source, offset)
    if (target) positions.set(target, offset + 3)
  })
  return positions
}
