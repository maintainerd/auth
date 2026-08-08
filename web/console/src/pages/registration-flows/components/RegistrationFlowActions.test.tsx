import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { RegistrationFlowActions } from "./RegistrationFlowActions"
import type { RegistrationFlow } from "@/services/api/registration-flows/types"

const { deleteMutateAsync, statusMutateAsync, navigateMock, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  deleteMutateAsync: vi.fn(),
  statusMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useDeleteRegistrationFlow: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateRegistrationFlowStatus: () => ({ mutateAsync: statusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeFlow(overrides: Partial<RegistrationFlow> = {}): RegistrationFlow {
  return {
    registration_flow_id: "f1",
    name: "Seller Signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "c1",
    verification_required: false,
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

async function openMenu() {
  await u().click(screen.getByRole("button", { name: /open menu/i }))
}

describe("RegistrationFlowActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("offers view, edit, deactivate and delete for an active regular flow", async () => {
    renderWithProviders(<RegistrationFlowActions registrationFlow={makeFlow()} />)
    await openMenu()

    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.getByText("Edit Registration Flow")).toBeInTheDocument()
    expect(screen.getByText("Deactivate Registration Flow")).toBeInTheDocument()
    expect(screen.getByText("Delete Registration Flow")).toBeInTheDocument()
    expect(screen.queryByText("Activate Registration Flow")).not.toBeInTheDocument()
  })

  it("offers activate for an inactive flow", async () => {
    renderWithProviders(<RegistrationFlowActions registrationFlow={makeFlow({ status: "inactive" })} />)
    await openMenu()

    expect(await screen.findByText("Activate Registration Flow")).toBeInTheDocument()
    expect(screen.queryByText("Deactivate Registration Flow")).not.toBeInTheDocument()
  })

  it("hides status changes and delete for a system flow", async () => {
    renderWithProviders(<RegistrationFlowActions registrationFlow={makeFlow({ is_system: true })} />)
    await openMenu()

    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.queryByText("Activate Registration Flow")).not.toBeInTheDocument()
    expect(screen.queryByText("Deactivate Registration Flow")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete Registration Flow")).not.toBeInTheDocument()
  })

  it("navigates to the details page", async () => {
    renderWithProviders(<RegistrationFlowActions registrationFlow={makeFlow()} />)
    await openMenu()
    await u().click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/registration-flows/f1")
  })

  it("deactivates the flow after confirmation", async () => {
    statusMutateAsync.mockResolvedValueOnce(makeFlow({ status: "inactive" }))
    renderWithProviders(<RegistrationFlowActions registrationFlow={makeFlow()} />)
    await openMenu()
    await u().click(await screen.findByText("Deactivate Registration Flow"))

    const dialog = await screen.findByRole("dialog")
    await u().click(await within(dialog).findByRole("button", { name: /deactivate/i }))

    await waitFor(() =>
      expect(statusMutateAsync).toHaveBeenCalledWith({
        registrationFlowId: "f1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Registration flow deactivated successfully")
  })
})
