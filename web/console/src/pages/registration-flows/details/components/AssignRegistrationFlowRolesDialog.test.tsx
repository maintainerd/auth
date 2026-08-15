import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { AssignRegistrationFlowRolesDialog } from "./AssignRegistrationFlowRolesDialog"

const { useRolesMock, assignMutateAsync, showSuccessMock, showErrorMock, onOpenChangeMock } = vi.hoisted(() => ({
  useRolesMock: vi.fn(),
  assignMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  onOpenChangeMock: vi.fn(),
}))

vi.mock("@/hooks/useRoles", () => ({
  useRoles: (...args: unknown[]) => useRolesMock(...args),
}))

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useAssignRegistrationFlowRoles: () => ({ mutateAsync: assignMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

const ROLES = [
  { role_id: "r1", name: "seller", description: "Seller role", is_default: false, is_system: false, status: "active", created_at: "", updated_at: "" },
  { role_id: "r2", name: "buyer", description: "Buyer role", is_default: false, is_system: false, status: "active", created_at: "", updated_at: "" },
]

// Roles the backend refuses for a public flow. Previously every fixture was
// active + non-system, so nothing proved the filter existed.
const SYSTEM_ROLE = { role_id: "r3", name: "super-admin", description: "System role", is_default: false, is_system: true, status: "active", created_at: "", updated_at: "" }
const INACTIVE_ROLE = { role_id: "r4", name: "retired", description: "Retired role", is_default: false, is_system: false, status: "inactive", created_at: "", updated_at: "" }

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

function renderDialog(props: Partial<Parameters<typeof AssignRegistrationFlowRolesDialog>[0]> = {}) {
  return renderWithProviders(
    <AssignRegistrationFlowRolesDialog
      open
      onOpenChange={onOpenChangeMock}
      registrationFlowId="f1"
      existingRoleIds={[]}
      {...props}
    />,
  )
}

describe("AssignRegistrationFlowRolesDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRolesMock.mockReturnValue({ data: { rows: ROLES, total: 2 }, isLoading: false })
  })

  it("only fetches roles while the dialog is open", () => {
    renderDialog({ open: false })
    expect(useRolesMock).toHaveBeenCalledWith(expect.any(Object), { enabled: false })

    vi.clearAllMocks()
    useRolesMock.mockReturnValue({ data: { rows: ROLES, total: 2 }, isLoading: false })
    renderDialog({ open: true })
    expect(useRolesMock).toHaveBeenCalledWith(expect.any(Object), { enabled: true })
  })

  it("lists assignable roles and hides the ones already assigned", () => {
    renderDialog({ existingRoleIds: ["r1"] })
    expect(screen.queryByText("seller")).not.toBeInTheDocument()
    expect(screen.getByText("buyer")).toBeInTheDocument()
  })

  it("assigns the selected roles and reports success", async () => {
    // Regression: the endpoint returns a bare array of roles. The service used to
    // read `.rows` off it, so a successful assignment threw and the user saw an
    // error toast for work that had actually been applied.
    assignMutateAsync.mockResolvedValueOnce([ROLES[0]])
    const user = u()
    renderDialog()

    await user.click(screen.getByRole("checkbox", { name: /seller/i }))
    await user.click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() =>
      expect(assignMutateAsync).toHaveBeenCalledWith({
        registrationFlowId: "f1",
        data: { role_ids: ["r1"] },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("1 role assigned successfully")
    expect(showErrorMock).not.toHaveBeenCalled()
    expect(onOpenChangeMock).toHaveBeenCalledWith(false)
  })

  it("assigns multiple roles and pluralizes the confirmation", async () => {
    assignMutateAsync.mockResolvedValueOnce(ROLES)
    const user = u()
    renderDialog()

    await user.click(screen.getByRole("checkbox", { name: /seller/i }))
    await user.click(screen.getByRole("checkbox", { name: /buyer/i }))
    await user.click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() =>
      expect(assignMutateAsync).toHaveBeenCalledWith({
        registrationFlowId: "f1",
        data: { role_ids: ["r1", "r2"] },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("2 roles assigned successfully")
  })

  it("keeps the dialog open and surfaces the error when assignment fails", async () => {
    const err = new Error("nope")
    assignMutateAsync.mockRejectedValueOnce(err)
    const user = u()
    renderDialog()

    await user.click(screen.getByRole("checkbox", { name: /seller/i }))
    await user.click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(showSuccessMock).not.toHaveBeenCalled()
    expect(onOpenChangeMock).not.toHaveBeenCalledWith(false)
  })

  it("disables the submit button until a role is selected", () => {
    renderDialog()
    expect(screen.getByRole("button", { name: /assign roles/i })).toBeDisabled()
  })

  it("filters the role list by the search query", async () => {
    const user = u()
    renderDialog()

    await user.type(screen.getByPlaceholderText(/search roles/i), "buy")

    expect(screen.getByText("buyer")).toBeInTheDocument()
    expect(screen.queryByText("seller")).not.toBeInTheDocument()
  })
})

// A registration flow is redeemed from a public link, so the backend refuses
// system roles, inactive roles, and roles carrying an administrative permission.
// The first two are visible client-side, so they must never be offered — an
// operator picking one would only discover the refusal on save.
describe("AssignRegistrationFlowRolesDialog — grantable-role cap", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRolesMock.mockReturnValue({
      data: { rows: [...ROLES, SYSTEM_ROLE, INACTIVE_ROLE], total: 4, page: 1, limit: 100, total_pages: 1 },
      isLoading: false,
    })
  })

  it("does not offer system roles", () => {
    renderDialog()
    expect(screen.getByRole("checkbox", { name: /seller/i })).toBeInTheDocument()
    expect(screen.queryByRole("checkbox", { name: /super-admin/i })).not.toBeInTheDocument()
  })

  it("does not offer inactive roles", () => {
    renderDialog()
    expect(screen.queryByRole("checkbox", { name: /retired/i })).not.toBeInTheDocument()
  })

  it("explains that only non-administrative roles can be assigned", () => {
    renderDialog()
    expect(screen.getByText(/only\s+non-administrative roles can be assigned/i)).toBeInTheDocument()
    expect(screen.getByText(/send an invite instead/i)).toBeInTheDocument()
  })

  it("Select All selects only the assignable roles", async () => {
    renderDialog()
    await u().click(screen.getByRole("button", { name: /select all/i }))

    // Both assignable roles are selected, and the refused ones were never
    // rendered to be selectable in the first place.
    expect(screen.getByRole("checkbox", { name: /seller/i })).toHaveAttribute("aria-checked", "true")
    expect(screen.getByRole("checkbox", { name: /buyer/i })).toHaveAttribute("aria-checked", "true")
    expect(screen.getAllByRole("checkbox")).toHaveLength(2)
  })

  it("still excludes already-assigned roles alongside the cap", () => {
    renderDialog({ existingRoleIds: ["r1"] })
    expect(screen.queryByRole("checkbox", { name: /seller/i })).not.toBeInTheDocument()
    expect(screen.getByRole("checkbox", { name: /buyer/i })).toBeInTheDocument()
  })
})
