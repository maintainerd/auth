import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ServiceHeader } from "./ServiceHeader"
import type { ServiceResponse } from "@/services/api/services/types"

const { navigateMock, deleteMutateAsync, updateStatusMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    deleteMutateAsync: vi.fn(),
    updateStatusMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useServices", () => ({
  useDeleteService: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateServiceStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeService(overrides: Partial<ServiceResponse> = {}): ServiceResponse {
  return {
    service_id: "s1", name: "auth-service", display_name: "Auth Service",
    description: "Handles authentication", version: "1.0.0", status: "active",
    is_system: false, api_count: 0, policy_count: 0,
    created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ServiceHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the display name, description, badges, and attributes", () => {
    renderWithProviders(<ServiceHeader service={makeService()} serviceId="s1" />)
    expect(screen.getByText("Auth Service")).toBeInTheDocument()
    expect(screen.getByText("Handles authentication")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("auth-service")).toBeInTheDocument()
    expect(screen.getByText("1.0.0")).toBeInTheDocument()
    expect(screen.getByText("Custom Service")).toBeInTheDocument()
  })

  it("navigates to edit with router state", async () => {
    renderWithProviders(<ServiceHeader service={makeService()} serviceId="s1" />)
    await u().click(screen.getByRole("button", { name: /edit/i }))
    expect(navigateMock).toHaveBeenCalledWith("/services/s1/edit", expect.objectContaining({
      state: expect.objectContaining({ from: "/services/s1" }),
    }))
  })

  it("offers the other statuses in the actions menu and confirms a change", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceHeader service={makeService({ status: "active" })} serviceId="s1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    expect(await screen.findByText("Set Maintenance")).toBeInTheDocument()
    expect(screen.getByText("Deprecate Service")).toBeInTheDocument()
    expect(screen.queryByText("Activate Service")).not.toBeInTheDocument()

    await u().click(screen.getByText("Deactivate Service"))
    await u().click(screen.getByRole("button", { name: "Deactivate Service" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        serviceId: "s1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Service status updated to inactive")
  })

  it("shows an error when the status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ServiceHeader service={makeService({ status: "inactive" })} serviceId="s1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Activate Service"))
    await u().click(screen.getByRole("button", { name: "Activate Service" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes the service, shows success, and navigates back", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceHeader service={makeService({ name: "auth-service" })} serviceId="s1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Service"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "auth-service")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("s1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Service deleted successfully")
    expect(navigateMock).toHaveBeenCalledWith("/services")
  })

  it("shows an error and does not navigate when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ServiceHeader service={makeService({ name: "auth-service" })} serviceId="s1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Service"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "auth-service")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("hides edit and the actions menu for system services", () => {
    renderWithProviders(<ServiceHeader service={makeService({ is_system: true })} serviceId="s1" />)
    expect(screen.getByText("System Service")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /edit/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })
})
