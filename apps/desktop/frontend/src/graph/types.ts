export type Vec3 = readonly [number, number, number]
export interface GraphNode {
  id: string
  path: string
  kind:
    | 'root'
    | 'directory'
    | 'source'
    | 'test'
    | 'config'
    | 'documentation'
    | 'asset'
    | 'tombstone'
  position: Vec3
}
export interface GraphEdge {
  source: string
  target: string
}
export interface GraphSnapshot {
  nodes: GraphNode[]
  edges: GraphEdge[]
}
