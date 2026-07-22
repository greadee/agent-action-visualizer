import { render, screen } from '@testing-library/react'
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
  })
})
