import { Component, type ErrorInfo, type ReactNode } from 'react'

interface ErrorBoundaryProps {
  children: ReactNode
  resetKey?: string
}

interface ErrorBoundaryState {
  hasError: boolean
}

/**
 * Top-level error boundary. Catches render/runtime errors anywhere in the route
 * tree and shows a recoverable fallback with a reload action instead of leaving
 * the user on a blank white screen. Self-contained (no hooks/providers) so it
 * still renders even when app context has failed to initialize.
 */
class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Surface the error for diagnostics; a real telemetry sink can hook in here.
    console.error('Unhandled application error:', error, info.componentStack)
  }

  componentDidUpdate(prevProps: ErrorBoundaryProps): void {
    if (this.state.hasError && prevProps.resetKey !== this.props.resetKey) {
      this.setState({ hasError: false })
    }
  }

  private readonly handleReload = (): void => {
    window.location.reload()
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div
          role="alert"
          data-console-auth-shell
          className="flex min-h-svh flex-col items-center justify-center gap-6 bg-background px-4 text-center text-foreground"
        >
          <div className="flex flex-col items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">Something went wrong</h1>
            <p className="max-w-sm text-sm text-muted-foreground">
              An unexpected error occurred. Reloading the page usually resolves it.
            </p>
          </div>
          <button
            type="button"
            onClick={this.handleReload}
            className="inline-flex h-9 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90"
          >
            Reload page
          </button>
        </div>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
