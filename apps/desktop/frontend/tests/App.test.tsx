import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from '../src/App'

vi.mock('@react-three/fiber', () => ({
  Canvas: () => <div data-testid="canvas" />,
}))

describe('App', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('identifies the local-only empty state', () => {
    render(<App />)
    expect(screen.getByText('Deterministic project hierarchy')).toBeTruthy()
    expect(screen.getByText(/Source stays on this machine/)).toBeTruthy()
    expect(screen.getByRole('searchbox', { name: 'SEARCH' })).toBeTruthy()
    expect(
      screen.getByRole('complementary', { name: 'Node inspector' }),
    ).toBeTruthy()
    expect(screen.getByLabelText('Graph legend')).toBeTruthy()
    expect(
      screen.getByRole('checkbox', { name: 'Render diagnostics' }),
    ).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: 'Access points' })).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add duration review batch' }),
    ).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add work review batch' }),
    ).toBeTruthy()
    expect(screen.getByText('SESSION ANALYTICS')).toBeTruthy()
    expect(screen.getByText('Current session view')).toBeTruthy()
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
    await waitFor(() => expectAccessCount('7 of 7 accesses'))
    fireEvent.click(screen.getByRole('button', { name: 'Work' }))
    expect(screen.getByText('+10043 / -5021')).toBeTruthy()
    expect(
      screen.getByText(/1 unknown, 1 binary, 0 unsupported, 0 pending work/),
    ).toBeTruthy()
    expect(
      screen.getByText('outward = additions · inward = deletions'),
    ).toBeTruthy()
  })

  it('advances deterministic focus and freezes the displayed state while paused', async () => {
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(
        screen.getAllByText('src/App.tsx', { selector: 'strong' }).length,
      ).toBeTruthy(),
    )
    expect(screen.getByText('patch · exact')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Pause updates' }))
    expect(screen.getByText('PAUSED')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() =>
      expect(
        screen.getAllByText('src/App.tsx', { selector: 'strong' }).length,
      ).toBeTruthy(),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Return live' }))
    await waitFor(() =>
      expect(
        screen.getAllByText('src/scene/GraphScene.tsx', {
          selector: 'strong',
        }).length,
      ).toBeTruthy(),
    )
    expect(screen.getByText('read · exact')).toBeTruthy()
  })

  it('switches between a bounded recent trail and the complete session', async () => {
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() => expectAccessCount('1 of 1 accesses'))
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() => expectAccessCount('2 of 2 accesses'))

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
    await waitFor(() => expectAccessCount('1 of 1 accesses'))
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() => expectAccessCount('2 of 2 accesses'))
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

  it('filters paths, operations, confidence, agents, and persists preferences', async () => {
    const { unmount } = render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() => expectAccessCount('1 of 1 accesses'))
    fireEvent.click(screen.getByRole('button', { name: 'Next review event' }))
    await waitFor(() => expectAccessCount('2 of 2 accesses'))

    fireEvent.change(
      screen.getByRole('searchbox', { name: 'PATH / DIRECTORY' }),
      {
        target: { value: 'App' },
      },
    )
    fireEvent.click(screen.getByRole('checkbox', { name: 'patch' }))
    fireEvent.click(screen.getByRole('checkbox', { name: 'exact' }))
    fireEvent.click(screen.getByRole('checkbox', { name: 'unknown' }))
    expectAccessCount('1 of 2 accesses')
    expect(screen.getByText(/Filtered session view/)).toBeTruthy()
    expect(screen.getAllByText('+0 / -0').length).toBeGreaterThan(0)
    expect(screen.getByText('Filtered from current view')).toBeTruthy()

    fireEvent.change(screen.getByRole('searchbox', { name: 'SEARCH' }), {
      target: { value: 'GraphScene' },
    })
    expect(
      screen.queryByRole('button', { name: 'src/scene/GraphScene.tsx' }),
    ).toBeNull()
    unmount()

    render(<App />)
    expect(
      screen.getByRole('searchbox', { name: 'PATH / DIRECTORY' }),
    ).toHaveProperty('value', 'App')
    fireEvent.click(screen.getByRole('button', { name: 'Reset filters' }))
    expect(
      screen.getByRole('searchbox', { name: 'PATH / DIRECTORY' }),
    ).toHaveProperty('value', '')
  })
})

function expectAccessCount(text: string) {
  expect(screen.getAllByText(text).length).toBeGreaterThan(0)
}
