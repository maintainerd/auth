import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import ApiDetailsPage from "./ApiDetailsPage"
import type { Api } from "@/services/api/api/types"

const { useApiMock, navigateMock, searchParamsRef } = vi.hoisted(() => ({
  useApiMock: vi.fn(),
  navigateMock: vi.fn(),
  searchParamsRef: { current: new URLSearchParams() },
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ apiId: "a1" }),
    useNavigate: () => navigateMock,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

vi.mock("@/hooks/useApis", () => ({
  useApi: (...args: unknown[]) => useApiMock(...args),
}))

vi.mock("./components", () => ({
  ApiHeader: ({ api }: { api: Api }) => <div data-testid="api-header">{api.display_name}</div>,
  ApiPermissionsTab: () => <div data-testid="permissions-tab" />,
}))

function makeApi(overrides: Partial<Api> = {}): Api {
  return {
    api_id: "a1", name: "billing-api", display_name: "Billing API",
    description: "Invoices", identifier: "https://api.example.com/billing",
    service: {
      service_id: "s1", name: "billing-service", display_name: "Billing Service",
      description: "", version: "1.0.0", status: "active", is_system: false,
      api_count: 0, policy_count: 0, created_at: "", updated_at: "",
    },
    status: "active", is_system: false, created_at: "", updated_at: "", ...overrides,
  }
}

describe("ApiDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
    useApiMock.mockReturnValue({ data: undefined, isLoading: false })
  })

  it("renders the loading state", () => {
    useApiMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<ApiDetailsPage />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the not-found state", () => {
    useApiMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ApiDetailsPage />)
    expect(screen.getByText("API not found")).toBeInTheDocument()
  })

  it("renders the API header with display name", () => {
    useApiMock.mockReturnValue({ data: makeApi(), isLoading: false })
    renderWithProviders(<ApiDetailsPage />)
    expect(screen.getByTestId("api-header")).toHaveTextContent("Billing API")
  })

  it("renders the permissions tab", () => {
    useApiMock.mockReturnValue({ data: makeApi(), isLoading: false })
    renderWithProviders(<ApiDetailsPage />)
    expect(screen.getByRole("tab", { name: "Permissions" })).toBeInTheDocument()
    expect(screen.getByTestId("permissions-tab")).toBeInTheDocument()
  })

  it("falls back to the permissions tab for an unknown ?tab value", () => {
    searchParamsRef.current = new URLSearchParams("tab=bogus")
    useApiMock.mockReturnValue({ data: makeApi(), isLoading: false })
    renderWithProviders(<ApiDetailsPage />)
    expect(screen.getByRole("tab", { name: "Permissions" })).toHaveAttribute("aria-selected", "true")
  })
})
