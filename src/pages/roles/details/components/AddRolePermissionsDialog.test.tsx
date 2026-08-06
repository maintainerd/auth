import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { AddRolePermissionsDialog } from "./AddRolePermissionsDialog"
import type { Api } from "@/services/api/api/types"
import type { PermissionEntity } from "@/services/api/permissions/types"

const {
  useApisMock,
  usePermissionsMock,
  addMutateAsync,
  addState,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  useApisMock: vi.fn(),
  usePermissionsMock: vi.fn(),
  addMutateAsync: vi.fn(),
  addState: { isPending: false },
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useApis", () => ({
  useApis: (...args: unknown[]) => useApisMock(...args),
}))

vi.mock("@/hooks/usePermissions", () => ({
  usePermissions: (...args: unknown[]) => usePermissionsMock(...args),
}))

vi.mock("@/hooks/useRoles", () => ({
  useAddRolePermissions: () => ({ mutateAsync: addMutateAsync, isPending: addState.isPending }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeApi(overrides: Partial<Api> = {}): Api {
  return {
    api_id: "a1",
    name: "orders-api",
    display_name: "Orders API",
    description: "Orders",
    identifier: "https://orders",
    status: "active",
    is_default: false,
    is_system: false,
    created_at: "",
    updated_at: "",
    ...overrides,
  } as Api
}

function makePermission(overrides: Partial<PermissionEntity> = {}): PermissionEntity {
  return {
    permission_id: "perm-read",
    name: "orders:read",
    description: "Read orders",
    api: makeApi(),
    status: "active",
    is_system: false,
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

function setApis(rows: Api[]) {
  useApisMock.mockReturnValue({ data: { rows, total: rows.length }, isLoading: false })
}

function setPermissions(rows: PermissionEntity[], overrides: Record<string, unknown> = {}) {
  usePermissionsMock.mockReturnValue({
    data: { rows, total: rows.length },
    isLoading: false,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

/** Search is debounced before it reaches the query, so assertions must outlast it. */
const DEBOUNCE_TIMEOUT = 3000

/**
 * Select-all is a real tri-state checkbox now, so it is queried by role
 * "checkbox" and its state is read off aria-checked rather than off a button
 * label that flipped between "Select All" and "Deselect All".
 */
const selectAll = () => screen.getByRole("checkbox", { name: /select all/i })

/** A permission row's own checkbox, named by the label the row renders. */
const permissionCheckbox = (name: string) =>
  screen.getByRole("checkbox", { name: new RegExp(name) })

const baseProps = {
  open: true,
  onOpenChange: vi.fn(),
  roleId: "r1",
  existingPermissionIds: [] as string[],
}

/** Opens the API combobox and picks the first API, which mounts the picker. */
async function selectApi(displayName = "Orders API") {
  const user = u()
  await user.click(screen.getByRole("combobox"))
  await user.click(await screen.findByText(displayName))
}

const WRITE_PERMS = [
  makePermission({ permission_id: "perm-write", name: "orders:write", description: "Write orders" }),
  makePermission({ permission_id: "perm-delete", name: "orders:delete", description: "Delete orders" }),
]

describe("AddRolePermissionsDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    addState.isPending = false
    setApis([makeApi()])
    setPermissions([])
  })

  it("issues no queries while the dialog is closed", () => {
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} open={false} />)
    // A closed dialog once fetched every API and permission in the tenant. The
    // API query is now registered but gated off; the permission query does not
    // exist at all until an API is chosen.
    expect(useApisMock).toHaveBeenCalledWith(expect.any(Object), { enabled: false })
    expect(usePermissionsMock).not.toHaveBeenCalled()
  })

  it("sends the API search term to the server instead of filtering the fetched page", async () => {
    // cmdk scored items against their `value` (the API UUID), so client-side
    // filtering matched nothing and API 101 was unreachable either way.
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await u().click(screen.getByRole("combobox"))
    await u().type(screen.getByPlaceholderText("Search APIs..."), "orders")

    await waitFor(
      () =>
        expect(useApisMock).toHaveBeenLastCalledWith(
          expect.objectContaining({ display_name: "orders", limit: 100 }),
          { enabled: true },
        ),
      { timeout: DEBOUNCE_TIMEOUT },
    )
  })

  it("tells the user when more APIs exist than the combobox page returned", async () => {
    useApisMock.mockReturnValue({
      data: { rows: [makeApi()], total: 412 },
      isLoading: false,
    })
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await u().click(screen.getByRole("combobox"))

    expect(await screen.findByText(/Showing the first 1 of 412 APIs/)).toBeInTheDocument()
  })

  it("reports a partial selection as mixed, not unchecked", async () => {
    // A ghost button could only say "Select All"; a tri-state box has to tell
    // assistive tech that some of the visible rows are already ticked.
    setPermissions([makePermission(), ...WRITE_PERMS])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    expect(selectAll()).toHaveAttribute("aria-checked", "false")
    await u().click(permissionCheckbox("orders:read"))
    expect(selectAll()).toHaveAttribute("aria-checked", "mixed")
    await u().click(selectAll())
    expect(selectAll()).toHaveAttribute("aria-checked", "true")
  })

  it("does not query permissions until an API is chosen", () => {
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    expect(useApisMock).toHaveBeenCalled()
    expect(usePermissionsMock).not.toHaveBeenCalled()
  })

  it("scopes the permission query to the chosen API", async () => {
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    expect(usePermissionsMock).toHaveBeenLastCalledWith(
      expect.objectContaining({ api_id: "a1", limit: 100, name: undefined }),
    )
  })

  it("sends the search term to the API instead of filtering the fetched page", async () => {
    // The permissions endpoint matches PermissionFilterDTO.Name with ILIKE, so
    // the term must reach the server or permission 101 stays unreachable.
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()
    await u().type(screen.getByPlaceholderText("Search permissions..."), "read")

    await waitFor(
      () =>
        expect(usePermissionsMock).toHaveBeenLastCalledWith(
          expect.objectContaining({ api_id: "a1", name: "read" }),
        ),
      { timeout: DEBOUNCE_TIMEOUT },
    )
  })

  it("select all never selects a permission that is not on screen", async () => {
    // The over-grant path: typing `read` used to leave Select All operating on
    // the unfiltered page, silently attaching every write/delete permission.
    setPermissions([makePermission(), ...WRITE_PERMS])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    // The server narrows the page to the single `read` match.
    setPermissions([makePermission()])
    await u().type(screen.getByPlaceholderText("Search permissions..."), "read")
    await waitFor(() => expect(screen.queryByText("orders:write")).not.toBeInTheDocument(), {
      timeout: DEBOUNCE_TIMEOUT,
    })

    await u().click(selectAll())

    expect(screen.getByText("1 permission selected")).toBeInTheDocument()
  })

  it("excludes permissions the role already holds from select all", async () => {
    setPermissions([makePermission(), ...WRITE_PERMS])
    renderWithProviders(
      <AddRolePermissionsDialog {...baseProps} existingPermissionIds={["perm-write", "perm-delete"]} />,
    )
    await selectApi()

    expect(screen.queryByText("orders:write")).not.toBeInTheDocument()
    await u().click(selectAll())
    expect(screen.getByText("1 permission selected")).toBeInTheDocument()
  })

  it("submits only the visible permissions select all picked", async () => {
    addMutateAsync.mockResolvedValue(undefined)
    const onOpenChange = vi.fn()
    setPermissions([makePermission(), ...WRITE_PERMS])
    renderWithProviders(
      <AddRolePermissionsDialog
        {...baseProps}
        onOpenChange={onOpenChange}
        existingPermissionIds={["perm-delete"]}
      />,
    )
    await selectApi()

    await u().click(selectAll())
    await u().click(screen.getByRole("button", { name: /add permissions/i }))

    await waitFor(() =>
      expect(addMutateAsync).toHaveBeenCalledWith({
        roleId: "r1",
        data: { permissions: ["perm-read", "perm-write"] },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("2 permissions added successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("toggles a single permission on and off", async () => {
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    await u().click(permissionCheckbox("orders:read"))
    expect(screen.getByText("1 permission selected")).toBeInTheDocument()
    await u().click(permissionCheckbox("orders:read"))
    expect(screen.queryByText("1 permission selected")).not.toBeInTheDocument()
  })

  it("deselect all clears the visible selection", async () => {
    setPermissions([makePermission(), ...WRITE_PERMS])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    await u().click(selectAll())
    expect(screen.getByText("3 permissions selected")).toBeInTheDocument()
    await u().click(selectAll())
    expect(screen.queryByText(/permissions selected/)).not.toBeInTheDocument()
  })

  it("tells the user when more matches exist than the page returned", async () => {
    usePermissionsMock.mockReturnValue({
      data: { rows: [makePermission()], total: 340 },
      isLoading: false,
    })
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    expect(screen.getByText(/Showing the first 1 of 340 permissions/)).toBeInTheDocument()
  })

  it("hides the refine hint when the page holds every match", async () => {
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    expect(screen.queryByText(/Showing the first/)).not.toBeInTheDocument()
  })

  it("shows the loading state while permissions load", async () => {
    setPermissions([], { data: undefined, isLoading: true })
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    expect(screen.getByText("Loading permissions...")).toBeInTheDocument()
  })

  it("shows the all-added empty state when nothing is selectable", async () => {
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} existingPermissionIds={["perm-read"]} />)
    await selectApi()

    expect(
      screen.getByText("All permissions for this API have already been added to the role"),
    ).toBeInTheDocument()
  })

  it("shows the no-match state when a search returns nothing", async () => {
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    setPermissions([])
    await u().type(screen.getByPlaceholderText("Search permissions..."), "zzz")

    await waitFor(
      () =>
        expect(screen.getByText("No permissions found matching your search")).toBeInTheDocument(),
      { timeout: DEBOUNCE_TIMEOUT },
    )
  })

  it("keeps the submit button disabled with no selection", () => {
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    expect(screen.getByRole("button", { name: /add permissions/i })).toBeDisabled()
  })

  it("shows an error when the mutation rejects", async () => {
    const err = new Error("fail")
    addMutateAsync.mockRejectedValueOnce(err)
    setPermissions([makePermission()])
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} />)
    await selectApi()

    await u().click(permissionCheckbox("orders:read"))
    await u().click(screen.getByRole("button", { name: /add permissions/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("cancel calls onOpenChange(false)", async () => {
    const onOpenChange = vi.fn()
    renderWithProviders(<AddRolePermissionsDialog {...baseProps} onOpenChange={onOpenChange} />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
