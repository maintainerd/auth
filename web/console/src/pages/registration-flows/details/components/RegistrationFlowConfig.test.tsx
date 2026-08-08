import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { RegistrationFlowConfig } from "./RegistrationFlowConfig"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

function makeFlow(overrides: Partial<RegistrationFlowDetail> = {}): RegistrationFlowDetail {
  return {
    registration_flow_id: "f1",
    name: "Seller Signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "c1",
    verification_required: true,
    required_fields: ["email", "phone"],
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("RegistrationFlowConfig", () => {
  it("renders the flow it is given without re-fetching it", () => {
    renderWithProviders(<RegistrationFlowConfig flow={makeFlow()} />)
    expect(screen.getByText("Required")).toBeInTheDocument()
    expect(screen.getByText("Email, Phone")).toBeInTheDocument()
  })

  it("labels the required fields in human form", () => {
    renderWithProviders(<RegistrationFlowConfig flow={makeFlow({ required_fields: ["fullname"] })} />)
    expect(screen.getByText("Full name")).toBeInTheDocument()
  })

  it("explains when only username and password are collected", () => {
    renderWithProviders(<RegistrationFlowConfig flow={makeFlow({ required_fields: [] })} />)
    expect(screen.getByText("Username and password only")).toBeInTheDocument()
  })

  it("reports verification as not required when the flow does not demand it", () => {
    renderWithProviders(<RegistrationFlowConfig flow={makeFlow({ verification_required: false })} />)
    expect(screen.getByText("Not required")).toBeInTheDocument()
  })

  it("cross-links the tenant-wide policy it overrides", () => {
    renderWithProviders(<RegistrationFlowConfig flow={makeFlow()} />)
    expect(screen.getByRole("link", { name: /security → registration/i })).toHaveAttribute(
      "href",
      "/security?tab=registration",
    )
  })
})
