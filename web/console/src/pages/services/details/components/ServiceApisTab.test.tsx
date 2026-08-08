import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ServiceApisTab } from "./ServiceApisTab"
import type { Api } from "@/services/api/api/types"

const { navigateMock, useServiceApisMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  useServiceApisMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("../hooks/useServiceApis", () => ({
  useServiceApis: (...args: unknown[]) => useServiceApisMock(...args),
}))

function makeApi(overrides: Partial<Api> = {}): Api {
  return {
    api_id: "a1",
    name: "billing-api",
    display_name: "Billing API",
    description: "Invoices and payments",
    identifier: "https://api.example.com/billing",
    service: {
      service_id: "s1", name: "auth-service", display_name: "Auth Service",
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

function response(rows: Api[], total = rows.length) {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

function getMenuButton() {
  return screen
    .getAllByRole("button", { name: /open menu/i })
    .find((el) => el.tagName === "BUTTON") as HTMLElement
}

describe("ServiceApisTab", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useServiceApisMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useServiceApisMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<ServiceApisTab serviceId="s1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useServiceApisMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ServiceApisTab serviceId="s1" />)
    expect(screen.getByText("Failed to load APIs")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useServiceApisMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<ServiceApisTab serviceId="s1" />)
    expect(screen.getByText("No APIs")).toBeInTheDocument()
  })

  it("renders an API row with name, identifier, and pagination", () => {
    useServiceApisMock.mockReturnValue({ data: response([makeApi()], 5), isLoading: false, isError: false })
    renderWithProviders(<ServiceApisTab serviceId="s1" />)
    expect(screen.getByText("Billing API")).toBeInTheDocument()
    expect(screen.getByText("billing-api")).toBeInTheDocument()
    expect(screen.getByText("https://api.example.com/billing")).toBeInTheDocument()
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })

  it("navigates to the API when the row body is clicked", async () => {
    const u = user()
    useServiceApisMock.mockReturnValue({ data: response([makeApi()]), isLoading: false, isError: false })
    renderWithProviders(<ServiceApisTab serviceId="s1" />)
    await u.click(screen.getByText("Billing API"))
    expect(navigateMock).toHaveBeenCalledWith("/apis/a1", expect.objectContaining({
      state: expect.objectContaining({ from: "/services/s1" }),
    }))
  })

  it("views an API via the menu", async () => {
    const u = user()
    useServiceApisMock.mockReturnValue({ data: response([makeApi()]), isLoading: false, isError: false })
    renderWithProviders(<ServiceApisTab serviceId="s1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/apis/a1", expect.anything())
  })
})
