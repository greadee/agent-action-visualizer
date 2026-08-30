import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from '../src/App'
import { demoGraph } from '../src/graph/demoGraph'

vi.mock('@react-three/fiber', () => ({
  Canvas: () => <div data-testid="canvas" />,
}))

describe('App', () => {
  beforeEach(() => {
    window.localStorage.clear()
    window.go = undefined
  })

  it('identifies the local-only empty state', () => {
    render(<App />)
    expect(screen.getByText('Choose your first repository')).toBeTruthy()
    expect(screen.getByText('Choose a local repository')).toBeTruthy()
    expect(screen.getByText(/Nothing is uploaded/)).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Choose folder…' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Setup & Help' })).toBeTruthy()
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
      screen.getByRole('checkbox', { name: 'Activity extrusions' }),
    ).toBeTruthy()
    expect(
      screen.getByText(/No recorded access or edit points yet/),
    ).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add duration review batch' }),
    ).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add work review batch' }),
    ).toBeTruthy()
    expect(
      screen.getByRole('button', { name: 'Add 1,000-event burst' }),
    ).toBeTruthy()
    expect(screen.getByText('SESSION ANALYTICS')).toBeTruthy()
    expect(screen.getByText('Current session view')).toBeTruthy()
    expect(screen.getByText('Absent — static scan')).toBeTruthy()
  })

  it('opens keyboard-accessible setup guidance for Codex and the generic wrapper', async () => {
    render(<App />)
    const helpButton = screen.getByRole('button', { name: 'Setup & Help' })
    fireEvent.click(helpButton)
    expect(
      screen.getByRole('dialog', { name: 'Setup & troubleshooting' }),
    ).toBeTruthy()
    expect(screen.getByText('Codex hook')).toBeTruthy()
    expect(screen.getByText('Generic wrapper')).toBeTruthy()
    expect(screen.getByText(/aav.exe codex install/)).toBeTruthy()
    expect(screen.getByText(/aav-wrapper.exe --project-root/)).toBeTruthy()
    expect(screen.getByText(/stable point on its file sphere/)).toBeTruthy()
    const closeButton = screen.getByRole('button', {
      name: 'Close setup and help',
    })
    await waitFor(() => expect(document.activeElement).toBe(closeButton))
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Tab' })
    expect(document.activeElement).toBe(closeButton)
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    await waitFor(() => expect(document.activeElement).toBe(helpButton))
  })

  it('uses the native picker, persists the repository, and reports an invalid path', async () => {
    const bridge = installDesktopBridge()
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Choose folder…' }))
    await waitFor(() => expect(bridge.pick).toHaveBeenCalledOnce())
    await waitFor(() => expect(screen.getByText('review-project')).toBeTruthy())
    expect(bridge.open).toHaveBeenCalledWith('C:\\review-project')
    expect(window.localStorage.getItem('aav.recentProject.v1')).toBe(
      'C:\\review-project',
    )

    bridge.open.mockRejectedValueOnce(
      new Error(
        'the project folder was not found; choose an existing repository',
      ),
    )
    fireEvent.change(screen.getByLabelText('PROJECT'), {
      target: { value: 'Z:\\missing' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Open path' }))
    expect((await screen.findByRole('alert')).textContent).toContain(
      'project folder was not found',
    )
  })

  it('reopens the recent project and restores Work mode', async () => {
    window.localStorage.setItem('aav.recentProject.v1', 'C:\\review-project')
    window.localStorage.setItem('aav.activityMode.v1', 'work')
    const bridge = installDesktopBridge()
    render(<App />)
    await waitFor(() =>
      expect(bridge.open).toHaveBeenCalledWith('C:\\review-project'),
    )
    expect(
      screen.getByRole('button', { name: 'Work' }).getAttribute('aria-pressed'),
    ).toBe('true')
    expect(screen.getByText('review-project')).toBeTruthy()
  })

  it('explains that the static graph survives a disconnected collector', async () => {
    installDesktopBridge({ collector: 'unavailable' })
    render(<App />)
    expect(
      await screen.findByText(
        'Collector disconnected · static graph available',
      ),
    ).toBeTruthy()
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
    await waitFor(() => expectAccessCount('7 of 7 accesses'), {
      timeout: 10_000,
    })
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

function installDesktopBridge(
  health: Record<string, string> = {
    status: 'ready',
    collector: 'ready',
    persistence: 'ready',
    recovery: 'none',
  },
) {
  const pick = vi.fn().mockResolvedValue('C:\\review-project')
  const open = vi.fn().mockResolvedValue({
    graph: demoGraph,
    root: 'C:\\review-project',
  })
  window.go = {
    main: {
      App: {
        Health: vi.fn().mockResolvedValue(health),
        LoadProject: vi.fn().mockResolvedValue(demoGraph),
        OpenProject: open,
        PickProjectDirectory: pick,
        PublishActivityEvent: vi.fn(),
        ListPersistedSessions: vi.fn().mockResolvedValue([]),
        ReplayPersistedSession: vi.fn(),
        Resync: vi.fn().mockResolvedValue({
          graph: { revision: 0, nodes: [], edges: [] },
        }),
      },
    },
  }
  return { open, pick }
}
