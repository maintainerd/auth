import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { ClientUris } from "./ClientUris"
import type { ClientResponse, Uri } from "@/services/api/clients/types"

function makeUri(overrides: Partial<Uri> = {}): Uri {
  return {
    uri_id: "u1",
    uri: "https://app.example.com/callback",
    type: "redirect_uri",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
    ...overrides,
  }
}

function makeClient(overrides: Partial<ClientResponse> = {}): ClientResponse {
  return {
    client_id: "c1", name: "console", display_name: "Console",
    client_type: "traditional", status: "active", is_default: false, is_system: false,
    allow_registration: true,
    allow_magic_link: false, created_at: "", updated_at: "", ...overrides,
  }
}

describe("ClientUris", () => {
  it("renders the empty state when the client has no URIs", () => {
    renderWithProviders(<ClientUris client={makeClient({ uris: [] })} />)
    expect(screen.getByText("No URIs or origins")).toBeInTheDocument()
  })

  it("groups URIs under their type sections with metadata", () => {
    const uris = [
      makeUri(),
      makeUri({ uri_id: "u2", uri: "https://app.example.com/login", type: "login_uri" }),
    ]
    renderWithProviders(<ClientUris client={makeClient({ uris })} />)

    expect(screen.getByText("Redirect URIs")).toBeInTheDocument()
    expect(screen.getByText("Login URI")).toBeInTheDocument()
    expect(screen.getByText("https://app.example.com/callback")).toBeInTheDocument()
    expect(screen.getByText("https://app.example.com/login")).toBeInTheDocument()
    expect(screen.getByText("u1")).toBeInTheDocument()
    expect(screen.getAllByText("Created").length).toBe(2)
    expect(screen.getAllByText("Last Updated").length).toBe(2)
  })

  it("still renders URI types outside the client type's capability set", () => {
    // A m2m client normally has no redirect URIs; stored records must still show.
    const uris = [makeUri()]
    renderWithProviders(<ClientUris client={makeClient({ client_type: "m2m", uris })} />)
    expect(screen.getByText("Redirect URIs")).toBeInTheDocument()
    expect(screen.getByText("https://app.example.com/callback")).toBeInTheDocument()
  })
})
