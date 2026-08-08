import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

import { renderWithProviders } from "@/test/utils"
import TenantAddOrUpdateForm from "./TenantAddOrUpdateForm"

const {
  useTenantByIdMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
  parseErrorMock,
} = vi.hoisted(() => ({
  useTenantByIdMock: vi.fn(),
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  parseErrorMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return {
    ...actual,
    useParams: () => ({}),
    useNavigate: () => navigateMock,
    useLocation: () => ({ state: null }),
  }
})

vi.mock("@/hooks/useTenants", () => ({
  useTenantById: (...args: unknown[]) => useTenantByIdMock(...args),
  useCreateTenant: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateTenant: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    parseError: parseErrorMock,
  }),
}))

/** Fills the form with input the client schema accepts, then submits. */
async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>, name = "acme-corp") {
  await user.type(screen.getByLabelText(/Tenant Name/i), name)
  await user.type(screen.getByLabelText(/Display Name/i), "Acme Corporation")
  await user.type(screen.getByLabelText(/Description/i), "The Acme tenant")
  await user.click(screen.getByRole("button", { name: "Create Tenant" }))
}

/** A flat service-layer error: message only, no field map. */
function serverError(message: string) {
  parseErrorMock.mockReturnValue({ message, fieldErrors: undefined, isValidationError: false })
  createMutateAsync.mockRejectedValue(new Error(message))
}

beforeEach(() => {
  vi.clearAllMocks()
  useTenantByIdMock.mockReturnValue({ data: undefined, isLoading: false })
})

describe("TenantAddOrUpdateForm status options", () => {
  it("offers pending, which the backend accepts", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TenantAddOrUpdateForm />)

    await user.click(screen.getByRole("combobox", { name: /Status/i }))

    for (const label of ["Active", "Inactive", "Pending", "Suspended"]) {
      expect(screen.getByRole("option", { name: label })).toBeInTheDocument()
    }
  })
})

describe("TenantAddOrUpdateForm server error mapping", () => {
  // "Name is reserved and cannot be used" (validation_tenant.go:49) only ever
  // matched the catch-all "name" keyword, so the user saw the raw string with no
  // hint about what to do.
  it("maps the reserved-name error to the name field with actionable copy", async () => {
    serverError("Name is reserved and cannot be used")
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TenantAddOrUpdateForm />)

    await fillAndSubmit(user, "acme-corp")

    await waitFor(() => {
      expect(screen.getByText(/reserved by the platform/i)).toBeInTheDocument()
    })
  })

  // The conflict message is `<name> tenant already exists`, which contains none
  // of the old keywords — it fell straight through to a generic toast.
  it("maps the duplicate-name conflict to the name field", async () => {
    serverError("acme-corp tenant already exists")
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TenantAddOrUpdateForm />)

    await fillAndSubmit(user)

    await waitFor(() => {
      expect(screen.getByText(/already taken/i)).toBeInTheDocument()
    })
  })

  it("maps the DNS-safe slug rejection to the name field", async () => {
    serverError(
      "Name must be a DNS-safe slug: lowercase letters, numbers, and hyphens, starting and ending with an alphanumeric",
    )
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TenantAddOrUpdateForm />)

    await fillAndSubmit(user)

    await waitFor(() => {
      expect(screen.getByText(/DNS-safe slug/i)).toBeInTheDocument()
    })
  })

  it("still maps the description length error to the description field", async () => {
    serverError("Description must be between 8 and 200 characters")
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TenantAddOrUpdateForm />)

    await fillAndSubmit(user)

    await waitFor(() => {
      expect(screen.getByText(/Description must be between 8 and 200/i)).toBeInTheDocument()
    })
  })
})

describe("TenantAddOrUpdateForm client-side validation", () => {
  it("blocks a reserved slug before it reaches the server", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TenantAddOrUpdateForm />)

    await fillAndSubmit(user, "console")

    await waitFor(() => {
      expect(screen.getByText(/reserved by the platform/i)).toBeInTheDocument()
    })
    expect(createMutateAsync).not.toHaveBeenCalled()
  })
})
