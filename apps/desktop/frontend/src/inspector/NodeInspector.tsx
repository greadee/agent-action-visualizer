import { useEffect, useState } from 'react'
import { normalizedIntervalDuration } from '../activity/timeExtrusions'
import {
  activityVisualCap,
  formatVisualCap,
  type ActivityDisplaySettings,
} from '../activity/displaySettings'
import type { ActivityMode } from '../activity/extrusions'
import type { GraphNode, TrailAccess } from '../graph/types'
import {
  formatDuration,
  formatExactDuration,
  formatTimestamp,
  formatWorkDelta,
} from './format'

function parentPath(path: string): string {
  const separator = path.lastIndexOf('/')
  return separator < 0 ? '.' : path.slice(0, separator)
}

export function NodeInspector({
  node,
  access,
  activityMode = 'time',
  activitySettings,
}: {
  node?: GraphNode
  access?: TrailAccess
  activityMode?: ActivityMode
  activitySettings?: ActivityDisplaySettings
}) {
  const [nowMs, setNowMs] = useState(() => Date.now())
  useEffect(() => {
    if (!access || access.ended_at) return
    const timer = window.setInterval(() => setNowMs(Date.now()), 250)
    return () => window.clearInterval(timer)
  }, [access])
  const accessDuration = access
    ? normalizedIntervalDuration(access, nowMs)
    : undefined
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
          {access && (
            <>
              <div className="rule" />
              <p className="panel__label">SELECTED ACCESS</p>
              <dl>
                <dt>Interval</dt>
                <dd>#{access.sequence}</dd>
                <dt>Duration</dt>
                <dd>{formatExactDuration(accessDuration)}</dd>
                <dt>State</dt>
                <dd>{access.ended_at ? 'Completed' : 'Active'}</dd>
                <dt>Operation</dt>
                <dd>{access.operations.at(-1) ?? 'access'}</dd>
                <dt>Work delta</dt>
                <dd>
                  {access.work_status === 'binary'
                    ? 'Binary file'
                    : access.work_status === 'unsupported_encoding'
                      ? 'Unsupported encoding'
                      : access.work_status === 'pending'
                        ? 'Calculating'
                        : formatWorkDelta(
                            access.lines_added,
                            access.lines_deleted,
                          )}
                </dd>
                <dt>Work evidence</dt>
                <dd>
                  {access.work_source ?? 'unknown'} ·{' '}
                  {access.work_confidence ?? 'inferred'}
                </dd>
                {activitySettings && (
                  <>
                    <dt>Visual scale</dt>
                    <dd>
                      {activitySettings.scale === 'log'
                        ? 'Logarithmic'
                        : 'Linear'}
                      ; cap{' '}
                      {formatVisualCap(
                        activityMode,
                        activityVisualCap(activityMode, activitySettings),
                      )}
                      ; exact values retained
                    </dd>
                  </>
                )}
                <dt>Evidence</dt>
                <dd>
                  {access.source} · {access.confidence}
                </dd>
              </dl>
            </>
          )}
        </>
      )}
    </aside>
  )
}
