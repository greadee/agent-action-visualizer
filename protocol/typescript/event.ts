export const schemaVersion = "1.0" as const;

export type SourceConfidence = "exact" | "correlated" | "observed" | "inferred";
export type SourceType =
  | "native_hook"
  | "structured_stream"
  | "editor"
  | "filesystem"
  | "git"
  | "wrapper"
  | "synthetic";
export type EventType =
  | "session_started"
  | "session_stopped"
  | "agent_started"
  | "agent_stopped"
  | "tool_started"
  | "tool_completed"
  | "file_focused"
  | "file_read"
  | "file_created"
  | "file_modified"
  | "file_patched"
  | "file_renamed"
  | "file_moved"
  | "file_deleted"
  | "directory_created"
  | "directory_renamed"
  | "directory_deleted"
  | "command_started"
  | "command_completed"
  | "access_interval_opened"
  | "access_interval_closed"
  | "diff_calculated"
  | "project_rescanned"
  | "adapter_connected"
  | "adapter_disconnected"
  | "adapter_warning"
  | "event_dropped_or_coalesced";

export interface NormalizedEvent {
  schema_version: typeof schemaVersion;
  event_id: string;
  event_type: EventType;
  source_type: SourceType;
  source_confidence: SourceConfidence;
  timestamp: string;
  session_id?: string;
  project_id?: string;
  agent_id?: string;
  agent_type?: string;
  adapter_id?: string;
  adapter_version?: string;
  operation?: string;
  status?: string;
  monotonic_timestamp?: number;
  correlation_id?: string;
  parent_event_id?: string;
  process_id?: number;
  thread_id?: string;
  turn_id?: string;
  tool_name?: string;
  command?: string;
  project_root?: string;
  path?: string;
  previous_path?: string;
  node_id?: string;
  is_directory?: boolean;
  is_binary?: boolean;
  duration_ms?: number;
  lines_added?: number;
  lines_deleted?: number;
  bytes_before?: number;
  bytes_after?: number;
  content_hash_before?: string;
  content_hash_after?: string;
  phase_id?: string;
  slice_id?: string;
  action_label?: string;
  metadata?: Record<string, unknown>;
  [futureField: string]: unknown;
}
