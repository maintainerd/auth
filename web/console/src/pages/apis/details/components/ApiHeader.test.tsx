import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ApiHeader } from "./ApiHeader"
import type { Api } from "@/services/api/api/types"

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

vi.mock("@/hooks/useApis", () => ({
  useDeleteApi: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateApiStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeApi(overrides: Partial<Api> = {}): Api {
  return {
    api_id: "a1", name: "billing-api", display_name: "Billing API",
    description: "Invoices and payments", identifier: "https://api.example.com/billing",
    service: {
      service_id: "s1", name: "billing-service", display_name: "Billing Service",
      description: "", version: "1.0.0", status: "active", is_system: false,
      api_count: 0, policy_count: 0, created_at: "", updated_at: "",
    },
    status: "active", is_system: false,
    created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ApiHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the display name, description, badges, and attributes", () => {
    renderWithProviders(<ApiHeader api={makeApi()} apiId="a1" />)
    expect(screen.getByText("Billing API")).toBeInTheDocument()
    expect(screen.getByText("Invoices and payments")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("billing-api")).toBeInTheDocument()
    expect(screen.getByText("https://api.example.com/billing")).toBeInTheDocument()
    expect(screen.getByText("Billing Service")).toBeInTheDocument()
    expect(screen.getByText("Custom API")).toBeInTheDocument()
  })

  it("navigates to edit with router state", async () => {
    renderWithProviders(<ApiHeader api={makeApi()} apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /edit/i }))
    expect(navigateMock).toHaveBeenCalledWith("/apis/a1/edit", expect.objectContaining({
      state: expect.objectContaining({ from: "/apis/a1" }),
    }))
  })

  it("offers the other status in the actions menu and confirms a change", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiHeader api={makeApi({ status: "active" })} apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    expect(await screen.findByText("Deactivate API")).toBeInTheDocument()
    expect(screen.queryByText("Activate API")).not.toBeInTheDocument()

    await u().click(screen.getByText("Deactivate API"))
    await u().click(screen.getByRole("button", { name: "Deactivate API" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        apiId: "a1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("API status updated to inactive")
  })

  it("shows an error when the status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ApiHeader api={makeApi({ status: "inactive" })} apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Activate API"))
    await u().click(screen.getByRole("button", { name: "Activate API" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes the API, shows success, and navigates back", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiHeader api={makeApi({ name: "billing-api" })} apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete API"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "billing-api")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("a1"))
    expect(showSuccessMock).toHaveBeenCalledWith("API deleted successfully")
    expect(navigateMock).toHaveBeenCalledWith("/apis")
  })

  it("shows an error and does not navigate when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ApiHeader api={makeApi({ name: "billing-api" })} apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete API"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "billing-api")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("hides edit and the actions menu for system APIs", () => {
    renderWithProviders(<ApiHeader api={makeApi({ is_system: true })} apiId="a1" />)
    expect(screen.getByText("System API")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /edit/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })
})
