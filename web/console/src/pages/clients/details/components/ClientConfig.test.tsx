import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { ClientConfig } from "./ClientConfig"

const { useClientConfigMock } = vi.hoisted(() => ({
  useClientConfigMock: vi.fn(),
}))

vi.mock("@/hooks/useClients", () => ({
  useClientConfig: (...args: unknown[]) => useClientConfigMock(...args),
}))

describe("ClientConfig", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useClientConfigMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useClientConfigMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<ClientConfig clientId="c1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useClientConfigMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ClientConfig clientId="c1" />)
    expect(screen.getByText("Failed to load configuration")).toBeInTheDocument()
  })

  it("renders the empty state when no known config keys exist", () => {
    useClientConfigMock.mockReturnValue({ data: {}, isLoading: false, isError: false })
    renderWithProviders(<ClientConfig clientId="c1" />)
    expect(screen.getByText("No configuration")).toBeInTheDocument()
  })

  it("renders grouped sections with formatted values", () => {
    useClientConfigMock.mockReturnValue({
      data: {
        grant_types: ["authorization_code", "refresh_token"],
        pkce_required: true,
        access_token_lifetime: 3600,
      },
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ClientConfig clientId="c1" />)
    expect(screen.getByText("Authorization Flow")).toBeInTheDocument()
    expect(screen.getByText("Token Settings")).toBeInTheDocument()
    expect(screen.getByText("grant_types")).toBeInTheDocument()
    expect(screen.getByText("authorization_code, refresh_token")).toBeInTheDocument()
    expect(screen.getByText("3600")).toBeInTheDocument()
  })
})
