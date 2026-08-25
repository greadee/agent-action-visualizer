import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { NodeInspector } from '../src/inspector/NodeInspector'
import type { GraphNode, TrailAccess } from '../src/graph/types'

const node: GraphNode = {
  id: 'file-a',
  path: 'src/a.ts',
  kind: 'source',
  position: [1, 2, 3],
}
const access: TrailAccess = {
  sequence: 7,
  node_id: node.id,
  path: node.path,
  started_at: '2026-07-23T12:00:00.000Z',
  ended_at: '2026-07-23T12:00:01.250Z',
  duration_ms: 1_250,
  operations: ['patch'],
  source: 'native_hook',
  confidence: 'exact',
  lines_added: 8,
  lines_deleted: 3,
  work_status: 'known',
  work_source: 'structured_patch',
  work_confidence: 'exact',
}

it('shows the selected access interval with its exact duration', () => {
  render(<NodeInspector node={node} access={access} />)
  expect(screen.getByText('SELECTED ACCESS')).toBeTruthy()
  expect(screen.getByText('1s 250ms')).toBeTruthy()
  expect(screen.getByText('Completed')).toBeTruthy()
  expect(screen.getByText('native_hook · exact')).toBeTruthy()
  expect(screen.getByText('+8 / -3')).toBeTruthy()
  expect(screen.getByText('structured_patch · exact')).toBeTruthy()
})
