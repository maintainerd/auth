import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PASSWORD_POLICY_DEFAULTS } from "@/lib/validations"
import PasswordPoliciesFormPage from "./PasswordPoliciesPage"

const { usePasswordPoliciesMock, updateMutateAsync, navigateMock } = vi.hoisted(() => ({
  usePasswordPoliciesMock: vi.fn(),
  updateMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/usePasswordPolicies", () => ({
  usePasswordPolicies: () => usePasswordPoliciesMock(),
  useUpdatePasswordPolicies: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
    parseError: (e: unknown) => ({ message: String(e), fieldErrors: {} }),
  }),
}))

const savedPolicy = {
  min_length: 20,
  max_length: 128,
  require_uppercase: true,
  require_lowercase: true,
  require_number: true,
  require_symbol: true,
  reject_common_passwords: true,
  check_hibp: true,
  password_history_count: 10,
  max_age_days: 90,
  temporary_password_validity_hours: 24,
  hash_algorithm: "argon2id",
  min_strength_score: 4,
}

beforeEach(() => vi.clearAllMocks())

describe("PasswordPoliciesFormPage", () => {
  // THE regression. The form used to render fully interactive over its DEFAULT
  // values while the GET was in flight, with Save enabled. Saving before hydration
  // overwrote the tenant's real policy with defaults — the PUT sends all 13 fields.
  it("does not render an editable form while the policy is still loading", () => {
    usePasswordPoliciesMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    renderWithProviders(<PasswordPoliciesFormPage />)

    expect(screen.queryByRole("button", { name: /save changes/i })).not.toBeInTheDocument()
    // And no field is present to type a default into.
    expect(screen.queryByLabelText(/minimum length/i)).not.toBeInTheDocument()
  })

  it("renders the saved policy once loaded, not the defaults", async () => {
    usePasswordPoliciesMock.mockReturnValue({ data: savedPolicy, isLoading: false, isError: false })
    renderWithProviders(<PasswordPoliciesFormPage />)

    expect(await screen.findByDisplayValue("20")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /save changes/i })).toBeInTheDocument()
  })

  it("shows an error state instead of the form when the load fails", () => {
    usePasswordPoliciesMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<PasswordPoliciesFormPage />)

    expect(screen.getByText(/failed to load password policy/i)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /save changes/i })).not.toBeInTheDocument()
  })

  // A config page edits rules that surface elsewhere, so it must show what they
  // produce.
  it("previews the requirements a user will be shown", async () => {
    usePasswordPoliciesMock.mockReturnValue({ data: savedPolicy, isLoading: false, isError: false })
    renderWithProviders(<PasswordPoliciesFormPage />)

    expect(await screen.findByText("Preview")).toBeInTheDocument()
    expect(screen.getByText(/at least 20 characters/i)).toBeInTheDocument()
  })

  // Restoring defaults must actually move the fields, not just mark the form
  // dirty — and it must leave the change unsaved so the operator can review it.
  it("restores the shipped defaults into the form without saving", async () => {
    const user = userEvent.setup()
    usePasswordPoliciesMock.mockReturnValue({ data: savedPolicy, isLoading: false, isError: false })
    renderWithProviders(<PasswordPoliciesFormPage />)

    expect(await screen.findByDisplayValue("20")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /restore recommended defaults/i }))

    await waitFor(() =>
      expect(screen.getByDisplayValue(String(PASSWORD_POLICY_DEFAULTS.min_length))).toBeInTheDocument(),
    )
    expect(updateMutateAsync).not.toHaveBeenCalled()
  })

  // Changing the hash algorithm behaves unlike every other field: it only
  // affects passwords hashed afterwards. The operator must be told before saving.
  it("warns only when the hashing algorithm is changed away from the saved one", async () => {
    const user = userEvent.setup()
    usePasswordPoliciesMock.mockReturnValue({ data: savedPolicy, isLoading: false, isError: false })
    renderWithProviders(<PasswordPoliciesFormPage />)

    expect(await screen.findByDisplayValue("20")).toBeInTheDocument()
    expect(screen.queryByText(/applies to new passwords only/i)).not.toBeInTheDocument()

    await user.click(screen.getByRole("combobox", { name: /hashing algorithm/i }))
    await user.click(await screen.findByRole("option", { name: /bcrypt/i }))

    expect(await screen.findByText(/applies to new passwords only/i)).toBeInTheDocument()
    // The warning must not claim users will be locked out — they are not.
    expect(screen.getByText(/existing users keep signing in normally/i)).toBeInTheDocument()
  })
})
