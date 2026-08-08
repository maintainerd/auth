import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ApiActions } from "./ApiActions"
import type { Api } from "@/services/api/api/types"

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

vi.mock("@/hooks/useApis", () => ({
  useDeleteApi: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateApiStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
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
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ApiActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("navigates to view details", async () => {
    renderWithProviders(<ApiActions api={makeApi()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/apis/a1")
  })

  it("navigates to edit", async () => {
    renderWithProviders(<ApiActions api={makeApi()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit API"))
    expect(navigateMock).toHaveBeenCalledWith("/apis/a1/edit")
  })

  it("deactivates an active API with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiActions api={makeApi({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate API"))
    await u().click(screen.getByRole("button", { name: "Deactivate API" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        apiId: "a1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("API status updated to inactive")
  })

  it("activates an inactive API with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiActions api={makeApi({ status: "inactive" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Activate API"))
    await u().click(screen.getByRole("button", { name: "Activate API" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        apiId: "a1",
        data: { status: "active" },
      }),
    )
  })

  it("shows error when status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ApiActions api={makeApi({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate API"))
    await u().click(screen.getByRole("button", { name: "Deactivate API" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a non-system API with confirmation", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiActions api={makeApi({ name: "billing-api" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete API"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "billing-api")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("a1"))
    expect(showSuccessMock).toHaveBeenCalledWith("API deleted successfully")
  })

  it("shows error when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ApiActions api={makeApi({ name: "billing-api" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete API"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "billing-api")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("hides edit, status, and delete actions for system APIs", async () => {
    renderWithProviders(<ApiActions api={makeApi({ is_system: true })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.queryByText("Edit API")).not.toBeInTheDocument()
    expect(screen.queryByText("Deactivate API")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete API")).not.toBeInTheDocument()
  })
})
