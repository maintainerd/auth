import type { ReactNode } from "react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import MFAPage from "./MFAPage"

const { fetchMFAStatusMock } = vi.hoisted(() => ({ fetchMFAStatusMock: vi.fn() }))

// AccountLayout is page chrome and pulls in the redux store; the subject here is
// which method rows render, so it is replaced with a passthrough.
vi.mock("@/components/layout/AccountLayout", () => ({
  default: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/services/api/mfa", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/api/mfa")>()),
  fetchMFAStatus: () => fetchMFAStatusMock(),
}))

function status(overrides: Record<string, unknown> = {}) {
  fetchMFAStatusMock.mockResolvedValue({
    is_totp_enabled: false,
    is_webauthn_enabled: false,
    is_sms_available: false,
    is_email_otp_available: false,
    backup_codes_count: 0,
    webauthn_keys: [],
    allowed_methods: ["totp", "webauthn", "backup_code"],
    mfa_required: false,
    ...overrides,
  })
}

// A factor the tenant disabled must not be offered. Showing it walks the user
// through an enrolment that is refused at the final step, which reads as a
// broken product rather than a policy decision.
describe("MFAPage — tenant policy filtering", () => {
  beforeEach(() => vi.clearAllMocks())

  it("hides methods the tenant does not allow", async () => {
    status({ allowed_methods: ["totp"] })
    renderWithProviders(<MFAPage />)

    expect(await screen.findByText("Authenticator app")).toBeInTheDocument()
    expect(screen.queryByText("Text message (SMS)")).not.toBeInTheDocument()
    expect(screen.queryByText("Email OTP")).not.toBeInTheDocument()
    expect(screen.queryByText("Passkeys")).not.toBeInTheDocument()
  })

  it("shows every method the tenant allows", async () => {
    status({ allowed_methods: ["totp", "sms", "email_otp", "webauthn"] })
    renderWithProviders(<MFAPage />)

    expect(await screen.findByText("Authenticator app")).toBeInTheDocument()
    expect(screen.getByText("Text message (SMS)")).toBeInTheDocument()
    expect(screen.getByText("Email OTP")).toBeInTheDocument()
    expect(screen.getByText("Passkeys")).toBeInTheDocument()
  })

  // If a tenant disables a factor someone already enrolled, hiding the row would
  // strand a live credential on their account with no way to reach and remove it.
  it("keeps an already-enrolled method visible after the tenant disables it", async () => {
    status({ allowed_methods: ["totp"], is_sms_available: true })
    renderWithProviders(<MFAPage />)

    expect(await screen.findByText("Text message (SMS)")).toBeInTheDocument()
  })

  // Fail closed: an unreadable policy offers nothing rather than everything.
  it("offers nothing when the policy is empty", async () => {
    status({ allowed_methods: [] })
    renderWithProviders(<MFAPage />)

    await screen.findByText("Authentication methods")
    expect(screen.queryByText("Authenticator app")).not.toBeInTheDocument()
    expect(screen.queryByText("Passkeys")).not.toBeInTheDocument()
  })
})
