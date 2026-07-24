import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import RegistrationFlowAddOrUpdateForm from "./RegistrationFlowAddOrUpdateForm"
import { useParams } from "react-router-dom"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

const {
  useRegistrationFlowMock,
  useRegistrationFlowRolesMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  useRegistrationFlowMock: vi.fn(),
  useRegistrationFlowRolesMock: vi.fn(),
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: vi.fn(() => ({})),
    useNavigate: () => navigateMock,
    useLocation: vi.fn(() => ({ state: null })),
  }
})

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useRegistrationFlow: (...args: unknown[]) => useRegistrationFlowMock(...args),
  useRegistrationFlowRoles: (...args: unknown[]) => useRegistrationFlowRolesMock(...args),
  useCreateRegistrationFlow: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateRegistrationFlow: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useClients", () => ({
  useClients: () => ({
    data: {
      rows: [
        { client_id: "c1", name: "storefront", display_name: "Storefront" },
        { client_id: "c2", name: "partner-portal", display_name: "Partner Portal" },
      ],
      total: 2,
    },
    isLoading: false,
  }),
  useClient: () => ({ data: undefined, isLoading: false }),
}))

vi.mock("@/hooks/useRoles", () => ({
  useRoles: () => ({
    data: {
      rows: [
        { role_id: "r1", name: "seller", description: "Seller role", is_default: false, is_system: false, status: "active", created_at: "", updated_at: "" },
        { role_id: "r2", name: "buyer", description: "Buyer role", is_default: false, is_system: false, status: "active", created_at: "", updated_at: "" },
      ],
      total: 2,
    },
    isLoading: false,
  }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    parseError: (error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }),
  }),
}))

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

// `/name/i` alone also matches the "Full name" required-field checkbox, and
// `/client/i` would also match nothing unless the combobox trigger is labelled
// (role="combobox" takes no name from its content).
const nameInput = () => screen.getByRole("textbox", { name: /^name/i })
const clientPicker = () => screen.getByRole("combobox", { name: /^client/i })

function setEditMode() {
  vi.mocked(useParams).mockReturnValue({ registrationFlowId: "f1" })
}

function setCreateMode() {
  vi.mocked(useParams).mockReturnValue({})
}

function makeFlow(overrides: Partial<RegistrationFlowDetail> = {}): RegistrationFlowDetail {
  return {
    registration_flow_id: "f1",
    name: "seller-signup",
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

function setFlowRoles(roleIds: string[]) {
  useRegistrationFlowRolesMock.mockReturnValue({
    data: {
      rows: roleIds.map((id) => ({
        role_id: id,
        name: id === "r1" ? "seller" : "buyer",
        description: "",
        is_default: false,
        is_system: false,
        status: "active",
        created_at: "",
        updated_at: "",
      })),
      total: roleIds.length,
      page: 1,
      limit: 100,
      total_pages: 1,
    },
    isLoading: false,
  })
}

describe("RegistrationFlowAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRegistrationFlowMock.mockReturnValue({ data: undefined, isLoading: false })
    setFlowRoles([])
    setCreateMode()
  })

  describe("status options", () => {
    it("offers only active and inactive — never draft (the backend rejects it with a 400)", async () => {
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await u().click(screen.getByRole("combobox", { name: /status/i }))

      const listbox = await screen.findByRole("listbox")
      expect(within(listbox).getByRole("option", { name: "Active" })).toBeInTheDocument()
      expect(within(listbox).getByRole("option", { name: "Inactive" })).toBeInTheDocument()
      expect(within(listbox).queryByRole("option", { name: /draft/i })).not.toBeInTheDocument()
      expect(within(listbox).getAllByRole("option")).toHaveLength(2)
    })
  })

  describe("create", () => {
    it("renders the create form", () => {
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)
      expect(screen.getAllByText("Create Registration Flow").length).toBeGreaterThanOrEqual(1)
      expect(nameInput()).toBeInTheDocument()
    })

    it("normalizes the name to a slug as you type, turning spaces into hyphens", async () => {
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      const name = nameInput()
      await u().type(name, "Seller Signup")

      // The name is the public link selector, so the field normalizes as you
      // type: spaces become hyphens rather than being silently dropped.
      expect(name).toHaveValue("seller-signup")
    })

    it("submits the slug name and the flow-behaviour fields, never an identifier", async () => {
      createMutateAsync.mockResolvedValueOnce(makeFlow({ registration_flow_id: "new-f" }))
      const user = u()
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await user.type(nameInput(), "seller-signup")
      await user.click(clientPicker())
      await user.click(await screen.findByRole("option", { name: "storefront" }))
      await user.click(screen.getByLabelText("Email"))
      await user.click(screen.getByRole("checkbox", { name: /seller/i }))
      await user.click(screen.getByRole("button", { name: /create registration flow/i }))

      await waitFor(() => expect(createMutateAsync).toHaveBeenCalled())
      const payload = createMutateAsync.mock.calls[0][0]
      expect(payload).not.toHaveProperty("identifier")
      expect(payload).toMatchObject({
        name: "seller-signup",
        status: "active",
        client_id: "c1",
        required_fields: ["email"],
        role_ids: ["r1"],
      })
      expect(showSuccessMock).toHaveBeenCalledWith("Registration flow created successfully")
      expect(navigateMock).toHaveBeenCalledWith("/registration-flows/new-f")
    })

    it("does not require a description", async () => {
      createMutateAsync.mockResolvedValueOnce(makeFlow())
      const user = u()
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await user.type(nameInput(), "seller-signup")
      await user.click(clientPicker())
      await user.click(await screen.findByRole("option", { name: "storefront" }))
      await user.click(screen.getByRole("button", { name: /create registration flow/i }))

      await waitFor(() => expect(createMutateAsync).toHaveBeenCalled())
      expect(screen.queryByText(/description is required/i)).not.toBeInTheDocument()
    })

    it("requires a name and a client", async () => {
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)
      await u().click(screen.getByRole("button", { name: /create registration flow/i }))

      await waitFor(() => expect(screen.getByText(/name is required/i)).toBeInTheDocument())
      expect(screen.getByText(/client is required/i)).toBeInTheDocument()
      expect(createMutateAsync).not.toHaveBeenCalled()
    })

    it("surfaces a backend error on the offending field", async () => {
      const err = new Error("registration flow with this name already exists")
      createMutateAsync.mockRejectedValueOnce(err)
      const user = u()
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await user.type(nameInput(), "seller-signup")
      await user.click(clientPicker())
      await user.click(await screen.findByRole("option", { name: "storefront" }))
      await user.click(screen.getByRole("button", { name: /create registration flow/i }))

      await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
      expect(screen.getByText(/already exists/i)).toBeInTheDocument()
    })
  })

  describe("edit hydration", () => {
    beforeEach(() => {
      setEditMode()
      useRegistrationFlowMock.mockReturnValue({ data: makeFlow(), isLoading: false })
      setFlowRoles(["r1"])
    })

    it("renders the loading skeleton while the flow is being fetched", () => {
      useRegistrationFlowMock.mockReturnValue({ data: undefined, isLoading: true })
      const { container } = renderWithProviders(<RegistrationFlowAddOrUpdateForm />)
      expect(container.querySelector(".animate-pulse")).toBeTruthy()
    })

    it("renders the loading skeleton until the assigned roles have resolved too", () => {
      useRegistrationFlowRolesMock.mockReturnValue({ data: undefined, isLoading: true })
      const { container } = renderWithProviders(<RegistrationFlowAddOrUpdateForm />)
      // Hydration must complete before the form is interactive, otherwise a
      // second reset could clobber an edit made in between.
      expect(container.querySelector(".animate-pulse")).toBeTruthy()
    })

    it("shows the not-found state for a missing flow", () => {
      useRegistrationFlowMock.mockReturnValue({ data: undefined, isLoading: false })
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)
      expect(screen.getByText("Registration flow not found")).toBeInTheDocument()
    })

    it("hydrates name, description, verification_required, required_fields and roles in one pass", async () => {
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await waitFor(() => expect(screen.getByDisplayValue("seller-signup")).toBeInTheDocument())
      expect(screen.getByDisplayValue("Sellers onboard here")).toBeInTheDocument()

      // verification_required = true must be reflected in the switch...
      expect(screen.getByRole("switch", { name: /require email verification/i })).toBeChecked()
      // ...required_fields must be checked...
      expect(screen.getByLabelText("Email")).toBeChecked()
      expect(screen.getByLabelText("Phone")).toBeChecked()
      expect(screen.getByLabelText("Full name")).not.toBeChecked()
      // ...and the assigned role must be selected.
      expect(screen.getByRole("checkbox", { name: /seller/i })).toHaveAttribute("aria-checked", "true")
      expect(screen.getByRole("checkbox", { name: /buyer/i })).toHaveAttribute("aria-checked", "false")
    })

    it("warns that renaming changes the flow's public registration link", async () => {
      const user = u()
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await waitFor(() => expect(screen.getByDisplayValue("seller-signup")).toBeInTheDocument())
      // No warning until the name actually changes.
      expect(screen.queryByText(/changes its registration link/i)).not.toBeInTheDocument()

      await user.clear(nameInput())
      await user.type(nameInput(), "seller-signup-v2")

      expect(screen.getByText(/changes its registration link/i)).toBeInTheDocument()
    })

    it("round-trips required_fields and verification_required untouched on save", async () => {
      // The regression: required_fields lived in component state that edit never
      // populated, so an unrelated edit posted `required_fields: []` (wiping the
      // flow's field requirements) and `verification_required: false` (silently
      // downgrading the verification policy).
      updateMutateAsync.mockResolvedValueOnce(makeFlow())
      const user = u()
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await waitFor(() => expect(screen.getByDisplayValue("seller-signup")).toBeInTheDocument())
      await user.clear(nameInput())
      await user.type(nameInput(), "seller-signup-v2")
      await user.click(screen.getByRole("button", { name: /update registration flow/i }))

      await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled())
      const { registrationFlowId, data } = updateMutateAsync.mock.calls[0][0]
      expect(registrationFlowId).toBe("f1")
      expect(data).not.toHaveProperty("identifier")
      expect(data).not.toHaveProperty("client_id")
      expect(data).toMatchObject({
        name: "seller-signup-v2",
        status: "active",
        verification_required: true,
        required_fields: ["email", "phone"],
        role_ids: ["r1"],
      })
    })

    it("submits an edited required_fields / verification_required / role selection", async () => {
      updateMutateAsync.mockResolvedValueOnce(makeFlow())
      const user = u()
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)

      await waitFor(() => expect(screen.getByDisplayValue("seller-signup")).toBeInTheDocument())
      await user.click(screen.getByLabelText("Phone"))
      await user.click(screen.getByRole("switch", { name: /require email verification/i }))
      await user.click(screen.getByRole("checkbox", { name: /buyer/i }))
      await user.click(screen.getByRole("button", { name: /update registration flow/i }))

      await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled())
      const { data } = updateMutateAsync.mock.calls[0][0]
      expect(data.verification_required).toBe(false)
      expect(data.required_fields).toEqual(["email"])
      expect(data.role_ids).toEqual(["r1", "r2"])
    })

    it("keeps the client fixed after creation", async () => {
      renderWithProviders(<RegistrationFlowAddOrUpdateForm />)
      await waitFor(() => expect(screen.getByDisplayValue("seller-signup")).toBeInTheDocument())
      expect(clientPicker()).toBeDisabled()
    })
  })
})
