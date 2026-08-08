import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { ClientMetadata } from "./ClientMetadata"

const { useClientConfigMock } = vi.hoisted(() => ({
  useClientConfigMock: vi.fn(),
}))

vi.mock("@/hooks/useClients", () => ({
  useClientConfig: (...args: unknown[]) => useClientConfigMock(...args),
}))

describe("ClientMetadata", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useClientConfigMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useClientConfigMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<ClientMetadata clientId="c1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useClientConfigMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ClientMetadata clientId="c1" />)
    expect(screen.getByText("Failed to load metadata")).toBeInTheDocument()
  })

  it("renders the empty state when only common config keys exist", () => {
    useClientConfigMock.mockReturnValue({
      data: { grant_types: ["authorization_code"] },
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ClientMetadata clientId="c1" />)
    expect(screen.getByText("No metadata")).toBeInTheDocument()
  })

  it("renders non-common entries with formatted values", () => {
    useClientConfigMock.mockReturnValue({
      data: { internal_tier: "gold", feature_flag: true },
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ClientMetadata clientId="c1" />)
    expect(screen.getByText("internal_tier")).toBeInTheDocument()
    expect(screen.getByText("gold")).toBeInTheDocument()
    expect(screen.getByText("feature_flag")).toBeInTheDocument()
    expect(screen.getByText("Enabled")).toBeInTheDocument()
  })
})
