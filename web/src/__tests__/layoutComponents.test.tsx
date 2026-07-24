import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { PageHeader } from '../components/layout/PageHeader'
import { MetricGrid } from '../components/layout/MetricGrid'
import { PageSection } from '../components/layout/PageSection'
import { ErrorBanner } from '../components/layout/ErrorBanner'
import { SkeletonRow } from '../components/layout/SkeletonRow'
import { PageSkeleton } from '../components/layout/PageSkeleton'

describe('PageHeader', () => {
  it('renders title', () => { render(<PageHeader title="Test" />); expect(screen.getByText('Test')).toBeTruthy() })
  it('renders subtitle', () => { render(<PageHeader title="T" subtitle="Sub" />); expect(screen.getByText('Sub')).toBeTruthy() })
  it('renders ok badge', () => { render(<PageHeader title="T" badge={{ text: 'LIVE', variant: 'ok' }} />); expect(screen.getByText('LIVE')).toBeTruthy() })
  it('renders err badge', () => { render(<PageHeader title="T" badge={{ text: 'HALTED', variant: 'err' }} />); expect(screen.getByText('HALTED')).toBeTruthy() })
  it('renders warn badge', () => { render(<PageHeader title="T" badge={{ text: 'OFFLINE', variant: 'warn' }} />); expect(screen.getByText('OFFLINE')).toBeTruthy() })
  it('renders actions', () => { render(<PageHeader title="T" actions={<button>Click</button>} />); expect(screen.getByText('Click')).toBeTruthy() })
})

describe('MetricGrid', () => {
  it('renders children', () => { render(<MetricGrid><span>Item</span></MetricGrid>); expect(screen.getByText('Item')).toBeTruthy() })
  it('defaults to 3 columns class', () => { const { container } = render(<MetricGrid><span /></MetricGrid>); expect(container.firstChild).toBeTruthy() })
  it('renders 5 column variant', () => { const { container } = render(<MetricGrid columns={5}><span /></MetricGrid>); expect(container.firstChild).toBeTruthy() })
})

describe('PageSection', () => {
  it('renders title', () => { render(<PageSection title="Section"><span /></PageSection>); expect(screen.getByText('Section')).toBeTruthy() })
  it('renders children', () => { render(<PageSection><span>Child</span></PageSection>); expect(screen.getByText('Child')).toBeTruthy() })
  it('renders error variant with border', () => { const { container } = render(<PageSection variant="error"><span /></PageSection>); expect(container.firstChild).toBeTruthy() })
  it('renders warning variant with border', () => { const { container } = render(<PageSection variant="warning"><span /></PageSection>); expect(container.firstChild).toBeTruthy() })
})

describe('ErrorBanner', () => {
  it('renders string error', () => { render(<ErrorBanner error="Something broke" />); expect(screen.getByText('Something broke')).toBeTruthy() })
  it('renders Error object', () => { render(<ErrorBanner error={new Error('Oops')} />); expect(screen.getByText('Oops')).toBeTruthy() })
  it('renders retry button', () => { render(<ErrorBanner error="Err" onRetry={() => {}} />); expect(screen.getByText('Retry')).toBeTruthy() })
  it('renders dismiss button', () => { render(<ErrorBanner error="Err" onDismiss={() => {}} />); expect(screen.getByText('Dismiss')).toBeTruthy() })
})

describe('SkeletonRow', () => {
  it('renders single row', () => { const { container } = render(<SkeletonRow />); expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0) })
  it('renders multiple rows', () => { const { container } = render(<SkeletonRow rows={3} />); expect(container.querySelectorAll('.animate-pulse').length).toBe(3) })
})

describe('PageSkeleton', () => {
  it('renders without crashing', () => { const { container } = render(<PageSkeleton />); expect(container.firstChild).toBeTruthy() })
})
