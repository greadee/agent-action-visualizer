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
    | 'generated'
    | 'tombstone'
  position: Vec3
  activity?: NodeActivity
}
export interface NodeActivity {
  group_key?: string
  last_commit?: string
  last_commit_at?: number
  last_event?: string
  access_count?: number
  total_time_ms?: number
  lines_added?: number
  lines_deleted?: number
  recent_tools?: string[]
  session_history?: string[]
  confidence?: string
}
export interface GraphEdge {
  source: string
  target: string
}
export interface GraphSnapshot {
  revision?: number
  nodes: GraphNode[]
  edges: GraphEdge[]
}
export interface GraphPatch {
  revision: number
  added: GraphNode[]
  updated: GraphNode[]
  removed: string[]
  edges: GraphEdge[]
}

export interface LiveFocusState {
  session_id: string
  active_node_id?: string
  active_path?: string
  previous_node_id?: string
  previous_path?: string
  secondary_node_ids?: string[]
  secondary_paths?: string[]
  operation?: string
  source?: string
  confidence?: string
  timestamp?: string
}

export interface ActivityEvent {
  schema_version: '1.0'
  event_id: string
  session_id: string
  source_type: string
  source_confidence: string
  event_type: string
  operation?: string
  timestamp: string
  path?: string
  metadata?: { secondary_paths?: string[] }
}
