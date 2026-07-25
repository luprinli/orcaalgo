import { Component, type ReactNode } from 'react'
import i18n from '../i18n/config'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: { componentStack: string }) {
    console.error('[ErrorBoundary]', error.message, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ border: '1px solid var(--trading-danger)', padding: 16 }}>
            <h3 style={{ color: 'var(--trading-danger)', marginBottom: 8 }}>
              {i18n.t('components:errorBoundary.somethingWentWrong', 'Something went wrong')}
            </h3>
            <p className="text-muted" style={{ fontSize: 12 }}>
              {this.state.error?.message}
            </p>
            <button
              className="btn btn-outline mt-2"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              {i18n.t('components:errorBoundary.retry', 'Retry')}
            </button>
          </div>
        )
      )
    }
    return this.props.children
  }
}

export default ErrorBoundary
