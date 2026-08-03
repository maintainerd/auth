import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'

interface ErrorBoundaryProps {
  children: ReactNode
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

  private readonly handleReload = (): void => {
    window.location.reload()
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div
          role="alert"
          data-auth-identity-shell
          className="flex min-h-svh flex-col items-center justify-center gap-6 px-4 text-center"
        >
          <div className="flex flex-col items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">Something went wrong</h1>
            <p className="max-w-sm text-sm text-muted-foreground">
              An unexpected error occurred. Reloading the page usually resolves it.
            </p>
          </div>
          <Button
            type="button"
            onClick={this.handleReload}
          >
            Reload page
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
