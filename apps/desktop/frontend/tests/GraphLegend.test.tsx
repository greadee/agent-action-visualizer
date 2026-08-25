import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DEFAULT_ACTIVITY_DISPLAY_SETTINGS } from '../src/activity/displaySettings'
import { GraphLegend } from '../src/scene/GraphLegend'

describe('GraphLegend shared UI integration', () => {
  it('uses the shared assistive label while keeping domain items local', () => {
    render(
      <GraphLegend
        activityMode="time"
        activitySettings={DEFAULT_ACTIVITY_DISPLAY_SETTINGS}
      />,
    )

    expect(screen.getByLabelText('Graph legend')).toBeTruthy()
    expect(screen.getByText('outward line = access duration')).toBeTruthy()
    expect(screen.getByText('current')).toBeTruthy()
    expect(screen.getByText('previous')).toBeTruthy()
    expect(screen.getByText(/Logarithmic visual scale/)).toBeTruthy()
  })
})
