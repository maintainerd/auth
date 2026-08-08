import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ServiceActions } from "./ServiceActions"
import type { Service } from "@/services/api/services/types"

const { navigateMock, updateStatusMutateAsync, deleteMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    updateStatusMutateAsync: vi.fn(),
    deleteMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({}),
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

function makeService(overrides: Partial<Service> = {}): Service {
  return {
    service_id: "s1",
    name: "auth-service",
    display_name: "Auth Service",
    description: "Handles authentication",
    version: "1.0.0",
    status: "active",
    is_system: false,
    api_count: 0,
    policy_count: 0,
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ServiceActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("navigates to view details", async () => {
    renderWithProviders(<ServiceActions service={makeService()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/services/s1")
  })

  it("navigates to edit", async () => {
    renderWithProviders(<ServiceActions service={makeService()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Service"))
    expect(navigateMock).toHaveBeenCalledWith("/services/s1/edit")
  })

  it("offers the other statuses but not the current one", async () => {
    renderWithProviders(<ServiceActions service={makeService({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("Set Maintenance")).toBeInTheDocument()
    expect(screen.getByText("Deprecate Service")).toBeInTheDocument()
    expect(screen.getByText("Deactivate Service")).toBeInTheDocument()
    expect(screen.queryByText("Activate Service")).not.toBeInTheDocument()
  })

  it("deactivates an active service with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceActions service={makeService({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Service"))
    await u().click(screen.getByRole("button", { name: "Deactivate Service" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        serviceId: "s1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Service status updated to inactive")
  })

  it("activates an inactive service with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceActions service={makeService({ status: "inactive" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Activate Service"))
    await u().click(screen.getByRole("button", { name: "Activate Service" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        serviceId: "s1",
        data: { status: "active" },
      }),
    )
  })

  it("shows error when status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ServiceActions service={makeService({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Service"))
    await u().click(screen.getByRole("button", { name: "Deactivate Service" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a non-system service with confirmation", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceActions service={makeService({ name: "auth-service" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Service"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "auth-service")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("s1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Service deleted successfully")
  })

  it("shows error when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ServiceActions service={makeService({ name: "auth-service" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Service"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "auth-service")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("hides edit, status, and delete actions for system services", async () => {
    renderWithProviders(<ServiceActions service={makeService({ is_system: true })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.queryByText("Edit Service")).not.toBeInTheDocument()
    expect(screen.queryByText("Set Maintenance")).not.toBeInTheDocument()
    expect(screen.queryByText("Deprecate Service")).not.toBeInTheDocument()
    expect(screen.queryByText("Deactivate Service")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete Service")).not.toBeInTheDocument()
  })
})
