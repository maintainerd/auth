import type { ReactElement } from "react"
import { render } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter, Routes, Route } from "react-router-dom"

/** A QueryClient with retries/logging disabled for deterministic tests. */
export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
}

interface RenderOptions {
  /** Initial URL — provides router params like tenantId / userPoolId. */
  route?: string
  /** Route pattern that `ui` is mounted under. */
  path?: string
  queryClient?: QueryClient
}

/**
 * Renders a component inside the providers it depends on:
 * QueryClientProvider + a MemoryRouter route (so useParams/useNavigate work).
 */
export function renderWithProviders(
  ui: ReactElement,
  { route = "/t1", path = "/:tenantId/*", queryClient = createTestQueryClient() }: RenderOptions = {},
) {
  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[route]}>
          <Routes>
            <Route path={path} element={ui} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    ),
  }
}
