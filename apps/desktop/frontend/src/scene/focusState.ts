import type { LiveFocusState } from '../graph/types'

export type FocusRole = 'current' | 'previous' | 'secondary' | 'older'

export function focusRoleForNode(
  nodeId: string,
  focus?: LiveFocusState,
  olderTrailNodeIds?: ReadonlySet<string>,
): FocusRole | undefined {
  if (nodeId === focus?.active_node_id) return 'current'
  if (nodeId === focus?.previous_node_id) return 'previous'
  if (focus?.secondary_node_ids?.includes(nodeId)) return 'secondary'
  if (olderTrailNodeIds?.has(nodeId)) return 'older'
  return undefined
}
