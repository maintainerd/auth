import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PolicyAssignDialog } from "./PolicyAssignDialog"
import type { Policy } from "@/services/api/policies/types"

const { usePoliciesMock, assignMutateAsync, assignState, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    usePoliciesMock: vi.fn(),
    assignMutateAsync: vi.fn(),
    assignState: { isPending: false },
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/usePolicies", () => ({
  usePolicies: (...args: unknown[]) => usePoliciesMock(...args),
}))

vi.mock("../hooks/useServicePolicyMutations", () => ({
  useServicePolicyMutations: () => ({
    assignPolicy: { mutateAsync: assignMutateAsync, isPending: assignState.isPending },
    removePolicy: { mutateAsync: vi.fn(), isPending: false },
  }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makePolicy(overrides: Partial<Policy> = {}): Policy {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

function setPolicies(rows: Policy[], overrides: Record<string, unknown> = {}) {
  usePoliciesMock.mockReturnValue({ data: { rows, total: rows.length }, isLoading: false, ...overrides })
}

/** Search is debounced before it reaches the query, so assertions must outlast it. */
const DEBOUNCE_TIMEOUT = 3000

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

/**
 * Select-all is a real tri-state checkbox now, so it is queried by role
 * "checkbox" and its state is read off aria-checked rather than off a button
 * label that flipped between "Select All" and "Deselect All".
 */
const selectAll = () => screen.getByRole("checkbox", { name: /select all/i })

/** A policy row's own checkbox, named by the label the row renders. */
const policyCheckbox = (name: string) =>
  screen.getByRole("checkbox", { name: new RegExp(name) })

const baseProps = {
  open: true,
  onOpenChange: vi.fn(),
  serviceId: "s1",
  existingPolicyIds: [] as string[],
}

describe("PolicyAssignDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    assignState.isPending = false
    setPolicies([])
  })

  it("disables the policies query while closed", () => {
    setPolicies([])
    renderWithProviders(<PolicyAssignDialog {...baseProps} open={false} />)
    expect(usePoliciesMock).toHaveBeenCalledWith(expect.anything(), { enabled: false })
  })

  it("shows the loading state while policies load", () => {
    usePoliciesMock.mockReturnValue({ data: undefined, isLoading: true })
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(screen.getByText("Loading policies...")).toBeInTheDocument()
  })

  it("shows the all-assigned empty state and excludes assigned policies", () => {
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} existingPolicyIds={["p1"]} />)
    expect(screen.getByText("All available policies are already assigned")).toBeInTheDocument()
    expect(screen.queryByText("read-only")).not.toBeInTheDocument()
  })

  it("sends the search term to the API instead of filtering the fetched page", async () => {
    // A client-side filter over a single limit:100 page made policy 101
    // unreachable; the term has to reach PolicyFilterDTO.Name to find it.
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    await u().type(screen.getByPlaceholderText("Search policies..."), "admin")

    await waitFor(
      () =>
        expect(usePoliciesMock).toHaveBeenLastCalledWith(
          expect.objectContaining({ name: "admin" }),
          { enabled: true },
        ),
      { timeout: DEBOUNCE_TIMEOUT },
    )
  })

  it("omits the name param when the search is empty", () => {
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(usePoliciesMock).toHaveBeenLastCalledWith(
      expect.objectContaining({ name: undefined }),
      { enabled: true },
    )
  })

  it("shows the no-match state when the server returns nothing for a search", async () => {
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    setPolicies([])
    await u().type(screen.getByPlaceholderText("Search policies..."), "zzz")

    await waitFor(
      () => expect(screen.getByText("No policies found matching your search")).toBeInTheDocument(),
      { timeout: DEBOUNCE_TIMEOUT },
    )
  })

  it("tells the user when more matches exist than the page returned", () => {
    usePoliciesMock.mockReturnValue({
      data: { rows: [makePolicy()], total: 250 },
      isLoading: false,
    })
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(
      screen.getByText(/Showing the first 1 of 250 policies/),
    ).toBeInTheDocument()
  })

  it("hides the refine hint when the page holds every match", () => {
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(screen.queryByText(/Showing the first/)).not.toBeInTheDocument()
  })

  it("select all only selects the rows on screen", async () => {
    // p2 is already assigned, so it is not rendered — and must not be selected.
    setPolicies([makePolicy(), makePolicy({ policy_id: "p2", name: "admin-policy" })])
    renderWithProviders(<PolicyAssignDialog {...baseProps} existingPolicyIds={["p2"]} />)

    await u().click(selectAll())
    expect(screen.getByText("1 policy selected")).toBeInTheDocument()
  })

  it("assigns only the rows select-all could see", async () => {
    assignMutateAsync.mockResolvedValue(undefined)
    setPolicies([makePolicy(), makePolicy({ policy_id: "p2", name: "admin-policy" })])
    renderWithProviders(<PolicyAssignDialog {...baseProps} existingPolicyIds={["p2"]} />)

    await u().click(selectAll())
    await u().click(screen.getByRole("button", { name: /assign policies/i }))

    await waitFor(() => expect(assignMutateAsync).toHaveBeenCalledTimes(1))
    expect(assignMutateAsync).toHaveBeenCalledWith("p1")
  })

  it("renders the system badge on policies", () => {
    setPolicies([makePolicy({ is_system: true })])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(screen.getByText("System")).toBeInTheDocument()
  })

  it("reports a partial selection as mixed, not unchecked", async () => {
    // A ghost button could only say "Select All"; a tri-state box has to tell
    // assistive tech that some of the visible rows are already ticked.
    setPolicies([makePolicy(), makePolicy({ policy_id: "p2", name: "admin-policy" })])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)

    expect(selectAll()).toHaveAttribute("aria-checked", "false")
    await u().click(policyCheckbox("read-only"))
    expect(selectAll()).toHaveAttribute("aria-checked", "mixed")
    await u().click(policyCheckbox("admin-policy"))
    expect(selectAll()).toHaveAttribute("aria-checked", "true")
  })

  it("toggles select-all and deselect-all", async () => {
    setPolicies([makePolicy(), makePolicy({ policy_id: "p2", name: "admin-policy" })])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    await u().click(selectAll())
    expect(screen.getByText("2 policies selected")).toBeInTheDocument()
    await u().click(selectAll())
    expect(screen.queryByText("2 policies selected")).not.toBeInTheDocument()
  })

  it("keeps the Assign button disabled with no selection", () => {
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(screen.getByRole("button", { name: /assign policies/i })).toBeDisabled()
  })

  it("assigns the selected policies and closes on success", async () => {
    assignMutateAsync.mockResolvedValue(undefined)
    const onOpenChange = vi.fn()
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} onOpenChange={onOpenChange} />)

    await u().click(policyCheckbox("read-only"))
    await u().click(screen.getByRole("button", { name: /assign policies/i }))

    await waitFor(() => expect(assignMutateAsync).toHaveBeenCalledWith("p1"))
    expect(showSuccessMock).toHaveBeenCalledWith("1 policy assigned successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("assigns multiple policies and shows the plural success message", async () => {
    assignMutateAsync.mockResolvedValue(undefined)
    setPolicies([makePolicy(), makePolicy({ policy_id: "p2", name: "admin-policy" })])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)

    await u().click(selectAll())
    await u().click(screen.getByRole("button", { name: /assign policies/i }))

    await waitFor(() => expect(assignMutateAsync).toHaveBeenCalledTimes(2))
    expect(showSuccessMock).toHaveBeenCalledWith("2 policies assigned successfully")
  })

  it("shows an error when assigning rejects", async () => {
    const err = new Error("fail")
    assignMutateAsync.mockRejectedValueOnce(err)
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)

    await u().click(policyCheckbox("read-only"))
    await u().click(screen.getByRole("button", { name: /assign policies/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("shows the Assigning... label while the mutation is pending", () => {
    assignState.isPending = true
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} />)
    expect(screen.getByText("Assigning...")).toBeInTheDocument()
  })

  it("cancel calls onOpenChange(false)", async () => {
    const onOpenChange = vi.fn()
    setPolicies([makePolicy()])
    renderWithProviders(<PolicyAssignDialog {...baseProps} onOpenChange={onOpenChange} />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
