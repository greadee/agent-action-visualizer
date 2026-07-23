import { describe, expect, it } from 'vitest'
import { focusRoleForNode } from '../src/scene/focusState'

describe('focusRoleForNode', () => {
  const focus = {
    session_id: 'review',
    active_node_id: 'current',
    previous_node_id: 'previous',
    secondary_node_ids: ['secondary', 'current'],
    trail: [],
  }

  it('distinguishes live focus roles with current taking priority', () => {
    expect(focusRoleForNode('current', focus)).toBe('current')
    expect(focusRoleForNode('previous', focus)).toBe('previous')
    expect(focusRoleForNode('secondary', focus)).toBe('secondary')
    expect(focusRoleForNode('unrelated', focus)).toBeUndefined()
  })

  it('marks older visible trail nodes after live roles', () => {
    const older = new Set(['older', 'secondary'])
    expect(focusRoleForNode('older', focus, older)).toBe('older')
    expect(focusRoleForNode('secondary', focus, older)).toBe('secondary')
  })
})
