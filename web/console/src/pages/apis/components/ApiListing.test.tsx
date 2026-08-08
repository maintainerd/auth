import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ApiListing } from "./ApiListing"
import type { Api } from "@/services/api/api/types"

const { useApisListMock, navigateMock } = vi.hoisted(() => ({
  useApisListMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({}),
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useApis", () => ({
  useApisList: (...args: unknown[]) => useApisListMock(...args),
  useDeleteApi: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateApiStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeApi(overrides: Partial<Api> = {}): Api {
  return {
    api_id: "a1",
    name: "billing-api",
    display_name: "Billing API",
    description: "Invoices and payments",
    identifier: "https://api.example.com/billing",
    service: {
      service_id: "s1", name: "billing-service", display_name: "Billing Service",
      description: "", version: "1.0.0", status: "active", is_system: false,
      api_count: 0, policy_count: 0, created_at: "", updated_at: "",
    },
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setApis(overrides: Record<string, unknown> = {}) {
  useApisListMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ApiListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setApis()
  })

  it("renders the loading state", () => {
    setApis({ isLoading: true })
    const { container } = renderWithProviders(<ApiListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows and navigates on row click", async () => {
    setApis({ data: { rows: [makeApi({ api_id: "a9", display_name: "Payments API" })], total: 1 } })
    renderWithProviders(<ApiListing />)
    const cell = screen.getByText("Payments API")
    await u().click(cell)
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/apis/a9"))
  })

  it("clicking New API navigates to the create route", async () => {
    renderWithProviders(<ApiListing />)
    const buttons = screen.getAllByRole("button", { name: /new api/i })
    await u().click(buttons[0])
    expect(navigateMock).toHaveBeenCalledWith("/apis/create")
  })
})
