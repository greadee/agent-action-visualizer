import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import App from '../src/App'

vi.mock('@react-three/fiber', () => ({
  Canvas: () => <div data-testid="canvas" />,
}))

describe('App', () => {
  it('identifies the local-only empty state', () => {
    render(<App />)
    expect(screen.getByText('Deterministic project hierarchy')).toBeTruthy()
    expect(screen.getByText(/Source stays on this machine/)).toBeTruthy()
    expect(screen.getByRole('searchbox', { name: 'SEARCH' })).toBeTruthy()
    expect(
      screen.getByRole('complementary', { name: 'Node inspector' }),
    ).toBeTruthy()
    expect(screen.getByLabelText('Graph legend')).toBeTruthy()
    expect(screen.getByText('Absent — static scan')).toBeTruthy()
  })

  it('advances deterministic focus and freezes the displayed state while paused', async () => {
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(
        screen.getByText('src/App.tsx', { selector: 'strong' }),
      ).toBeTruthy(),
    )
    expect(screen.getByText('patch · exact')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Pause updates' }))
    expect(screen.getByText('PAUSED')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(
        screen.getByText('src/App.tsx', { selector: 'strong' }),
      ).toBeTruthy(),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Return live' }))
    await waitFor(() =>
      expect(
        screen.getByText('src/scene/GraphScene.tsx', { selector: 'strong' }),
      ).toBeTruthy(),
    )
    expect(screen.getByText('read · exact')).toBeTruthy()
  })
})
