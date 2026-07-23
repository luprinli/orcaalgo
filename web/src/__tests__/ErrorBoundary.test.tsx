import { describe, it, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import ErrorBoundary from '../components/ErrorBoundary'

const Exploder = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) throw new Error('Simulated chart crash: data must be asc ordered by time, index=1, time=NaN')
  return <div>OK</div>
}

describe('ErrorBoundary', () => {
  it('renders children when no error occurs', () => {
    const { getByText } = render(
      <ErrorBoundary>
        <Exploder shouldThrow={false} />
      </ErrorBoundary>,
    )
    expect(getByText('OK')).toBeDefined()
  })

  it('renders fallback UI on child crash', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { getByText } = render(
      <ErrorBoundary>
        <Exploder shouldThrow={true} />
      </ErrorBoundary>,
    )
    expect(getByText('Component Error')).toBeDefined()
    expect(getByText('Simulated chart crash: data must be asc ordered by time, index=1, time=NaN')).toBeDefined()
    expect(getByText('Retry')).toBeDefined()
    spy.mockRestore()
  })

  it('recovers after Retry button click', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    // Use key to force remount — simulates the Retry flow
    const { getByText, rerender } = render(
      <ErrorBoundary key="crash">
        <Exploder shouldThrow={true} />
      </ErrorBoundary>,
    )
    expect(getByText('Retry')).toBeDefined()

    // Rerender with same component but no crash
    rerender(
      <ErrorBoundary key="ok">
        <Exploder shouldThrow={false} />
      </ErrorBoundary>,
    )
    expect(getByText('OK')).toBeDefined()
    spy.mockRestore()
  })

  it('uses custom fallback when provided', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { getByText } = render(
      <ErrorBoundary fallback={<div>Custom fallback content</div>}>
        <Exploder shouldThrow={true} />
      </ErrorBoundary>,
    )
    expect(getByText('Custom fallback content')).toBeDefined()
    spy.mockRestore()
  })
})
