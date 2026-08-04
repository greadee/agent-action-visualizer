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
    expect(screen.getByRole('checkbox', { name: 'Access points' })).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add duration review batch' }),
    ).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add work review batch' }),
    ).toBeTruthy()
    expect(screen.getByText('Absent — static scan')).toBeTruthy()
  })

  it('uses the shared exclusive control for activity mode', () => {
    render(<App />)
    const time = screen.getByRole('button', { name: 'Time' })
    const work = screen.getByRole('button', { name: 'Work' })

    expect(time.getAttribute('aria-pressed')).toBe('true')
    expect(work.getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(work)
    expect(time.getAttribute('aria-pressed')).toBe('false')
    expect(work.getAttribute('aria-pressed')).toBe('true')
    expect(
      screen.getByRole('combobox', { name: 'Activity visual scale' }),
    ).toHaveProperty('value', 'log')
    expect(
      screen.getByRole('combobox', { name: 'Work visual cap' }),
    ).toHaveProperty('value', '1000')
    fireEvent.change(
      screen.getByRole('combobox', { name: 'Activity visual scale' }),
      {
        target: { value: 'linear' },
      },
    )
    expect(screen.getByText(/Linear visual scale/)).toBeTruthy()
  })

  it('loads deterministic addition, deletion, mixed, and unknown work review states', async () => {
    render(<App />)
    fireEvent.click(
      screen.getByRole('button', { name: 'Add work review batch' }),
    )
    await waitFor(() =>
      expect(screen.getByText('7 of 7 accesses')).toBeTruthy(),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Work' }))
    expect(
      screen.getByText('outward = additions · inward = deletions'),
    ).toBeTruthy()
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

  it('switches between a bounded recent trail and the complete session', async () => {
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(screen.getByText('1 of 1 accesses')).toBeTruthy(),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(screen.getByText('2 of 2 accesses')).toBeTruthy(),
    )

    const limit = screen.getByRole('spinbutton', {
      name: 'Recent trail access limit',
    })
    expect(limit).toHaveProperty('value', '12')
    fireEvent.change(limit, { target: { value: '2' } })
    expect(limit).toHaveProperty('value', '2')

    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Complete session trail' }),
    )
    expect(
      screen.queryByRole('spinbutton', { name: 'Recent trail access limit' }),
    ).toBeNull()
    expect(screen.getByText('session travel → newest')).toBeTruthy()
  })

  it('replays a persisted session without mixing it into live focus', async () => {
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(screen.getByText('1 of 1 accesses')).toBeTruthy(),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(screen.getByText('2 of 2 accesses')).toBeTruthy(),
    )
    const sessions = screen.getByRole('combobox', {
      name: 'Persisted session',
    })
    fireEvent.focus(sessions)
    await waitFor(() =>
      expect(screen.getByRole('option', { name: /p4-review/ })).toBeTruthy(),
    )
    fireEvent.change(sessions, { target: { value: 'p4-review' } })
    await waitFor(() => expect(screen.getByText('REPLAY')).toBeTruthy())
    expect(screen.getByText('Event 3 of 3', { exact: false })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Previous access' }))
    await waitFor(() =>
      expect(screen.getByText('Event 2 of 3', { exact: false })).toBeTruthy(),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Return live' }))
    await waitFor(() => expect(screen.getByText('LIVE')).toBeTruthy())
  })
})
