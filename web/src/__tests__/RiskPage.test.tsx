import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../api/client', () => ({
  risk: {
    status: vi.fn().mockResolvedValue({ halted: false, equity: 100000, balance: 100000 }),
  },
  orders: { list: vi.fn(), cancel: vi.fn() },
}))

vi.mock('../hooks/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

import RiskPage from '../pages/RiskPage'

describe('RiskPage', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ history: [] }),
    })
  })

  it('renders without crashing', () => {
    render(
      <MemoryRouter>
        <RiskPage />
      </MemoryRouter>,
    )
    // Component renders, even if content is async. Just verify no unmounted error.
    expect(document.body).toBeDefined()
  })
})
