import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { ClientInformation } from "./ClientInformation"
import type { ClientResponse } from "@/services/api/clients/types"

function makeClient(overrides: Partial<ClientResponse> = {}): ClientResponse {
  return {
    client_id: "c1", name: "console", display_name: "Console",
    client_type: "traditional", status: "active", is_default: false, is_system: false,
    allow_registration: true,
    allow_magic_link: false, created_at: "", updated_at: "", ...overrides,
  }
}

describe("ClientInformation", () => {
  it("shows inherit labels when overrides are unset", () => {
    renderWithProviders(<ClientInformation client={makeClient()} />)
    expect(screen.getByText("Security & Sessions")).toBeInTheDocument()
    expect(screen.getAllByText("Inherits tenant default").length).toBe(4)
  })

  it("formats PKCE, ACR, and session timeouts when set", () => {
    renderWithProviders(
      <ClientInformation
        client={makeClient({
          require_pkce: true,
          required_acr: "2",
          session_idle_timeout: 1800,
          session_absolute_timeout: 86400,
        })}
      />,
    )
    expect(screen.getByText("Required")).toBeInTheDocument()
    expect(screen.getByText("Step-up — MFA (ACR 2)")).toBeInTheDocument()
    expect(screen.getByText("1800s · 30 min")).toBeInTheDocument()
    expect(screen.getByText("86400s · 1 day")).toBeInTheDocument()
  })

  it("formats disabled PKCE, hour-based timeouts, and raw seconds", () => {
    renderWithProviders(
      <ClientInformation
        client={makeClient({
          require_pkce: false,
          session_idle_timeout: 3600,
          session_absolute_timeout: 90,
        })}
      />,
    )
    expect(screen.getByText("Disabled")).toBeInTheDocument()
    expect(screen.getByText("3600s · 1 hour")).toBeInTheDocument()
    expect(screen.getByText("90s")).toBeInTheDocument()
  })
})
