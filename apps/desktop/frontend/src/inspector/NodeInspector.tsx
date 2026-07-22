import type { GraphNode } from '../graph/types'

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
          <dl>
            <dt>Node type</dt>
            <dd>{node.kind}</dd>
            <dt>Parent folder</dt>
            <dd>{parentPath(node.path)}</dd>
            <dt>Status</dt>
            <dd>Indexed</dd>
            <dt>Access count</dt>
            <dd>Not observed</dd>
            <dt>Total time</dt>
            <dd>Not observed</dd>
            <dt>Lines added / deleted</dt>
            <dd>Not observed</dd>
            <dt>Event confidence</dt>
            <dd title="No live adapter event is attached to this static node">
              Absent — static scan
            </dd>
            <dt>Recent tools</dt>
            <dd>None recorded</dd>
            <dt>Rename history</dt>
            <dd>None recorded</dd>
            <dt>Last event</dt>
            <dd>Not observed</dd>
          </dl>
        </>
      )}
    </aside>
  )
}
