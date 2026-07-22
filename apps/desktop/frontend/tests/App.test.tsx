import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import App from '../src/App'

vi.mock('@react-three/fiber', () => ({
  Canvas: () => <div data-testid="canvas" />,
}))

describe('App', () => {
  it('identifies the local-only empty state', () => {
    render(<App />)
    expect(
      screen.getByText('Open a repository to build its graph'),
    ).toBeTruthy()
    expect(screen.getByText(/No AI API or telemetry/)).toBeTruthy()
  })
})
