import type { GraphNode } from '../graph/types'
import { formatDuration, formatTimestamp } from './format'

function parentPath(path: string): string {
  const separator = path.lastIndexOf('/')
  return separator < 0 ? '.' : path.slice(0, separator)
}

export function NodeInspector({ node }: { node?: GraphNode }) {
  return (
    <aside className="inspector" aria-label="Node inspector">
      <p className="panel__label">INSPECTOR</p>
      {!node ? (
        <p className="muted">Select a node to inspect it.</p>
      ) : (
        <>
          <h2>{node.path}</h2>
          {node.activity?.group_key && (
            <p className="commit-group" title={node.activity.group_key}>
              Latest involvement · {node.activity.group_key.slice(0, 8)}
            </p>
          )}
          <dl>
            <dt>Node type</dt>
            <dd>{node.kind}</dd>
            <dt>Parent folder</dt>
            <dd>{parentPath(node.path)}</dd>
            <dt>Status</dt>
            <dd>
              {node.activity?.last_commit ? 'Git tracked' : 'Uncommitted'}
            </dd>
            <dt>Access count from Git</dt>
            <dd>{node.activity?.access_count ?? 0} commit touches</dd>
            <dt>Total time reported by agent</dt>
            <dd>{formatDuration(node.activity?.total_time_ms)}</dd>
            <dt>Lines added / deleted</dt>
            <dd>
              +{node.activity?.lines_added ?? 0} / -
              {node.activity?.lines_deleted ?? 0}
            </dd>
            <dt>Event confidence</dt>
            <dd title="Git values are inferred from commits; time and work values come from the agent report">
              {node.activity?.confidence ?? 'Absent — static scan'}
            </dd>
            <dt>Recent tools</dt>
            <dd>
              {node.activity?.recent_tools?.join(', ') || 'None recorded'}
            </dd>
            <dt>Session history</dt>
            <dd>
              {node.activity?.session_history?.join(', ') ||
                'Not reported by agent'}
            </dd>
            <dt>Rename history</dt>
            <dd>Not inferred from Git</dd>
            <dt>Last event</dt>
            <dd>{node.activity?.last_event ?? 'Not observed'}</dd>
            <dt>Last involvement</dt>
            <dd>{formatTimestamp(node.activity?.last_commit_at)}</dd>
          </dl>
        </>
      )}
    </aside>
  )
}
