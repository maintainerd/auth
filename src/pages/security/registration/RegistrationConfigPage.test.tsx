/**
 * The CAPTCHA on Signup switch is the only escape hatch out of a total signup
 * outage: the backend rejects every tokenless registration once a tenant has
 * captcha_on_signup persisted true and CAPTCHA_SECRET is configured, and the
 * hosted signup form does not render a challenge yet. The control was once
 * hardcoded `checked={false} onCheckedChange={() => {}} disabled`, so an
 * affected tenant saw "off" while registration was failing and had no way to
 * clear the flag. These cases pin the escape hatch: the persisted value must be
 * displayed, and a persisted true must remain switchable to false.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import RegistrationConfigPage from "./RegistrationConfigPage"
import type { RegistrationConfig } from "@/services/api/registration-config/types"

const { navigateMock, updateMutateAsync, showSuccessMock, showErrorMock, parseErrorMock, useRegistrationConfigMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    updateMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
    parseErrorMock: vi.fn(() => ({ message: "", fieldErrors: undefined })),
    useRegistrationConfigMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useRegistrationConfig", () => ({
  useRegistrationConfig: () => useRegistrationConfigMock(),
  useUpdateRegistrationConfig: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock, parseError: parseErrorMock }),
}))

function makeConfig(overrides: Partial<RegistrationConfig> = {}): RegistrationConfig {
  return {
    self_registration_enabled: true,
    require_email_verification: true,
    require_phone_verification: false,
    auto_confirm_enabled: false,
    verification_token_ttl_hours: 24,
    captcha_on_signup: false,
    registration_rate_limit_per_ip_per_hour: 10,
    allowed_email_domains: [],
    blocked_email_domains: [],
    ...overrides,
  }
}

/** Drives the page with a persisted config, as the API would return it. */
function renderWithConfig(config: RegistrationConfig) {
  useRegistrationConfigMock.mockReturnValue({ data: config, isLoading: false, isError: false })
  return renderWithProviders(<RegistrationConfigPage />, { route: "/security/registration/configure" })
}

const captchaSwitch = () => screen.getByRole("switch", { name: /captcha on signup/i })
const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("RegistrationConfigPage — CAPTCHA on Signup", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    parseErrorMock.mockReturnValue({ message: "", fieldErrors: undefined })
  })

  it("shows the switch ON and interactive when the tenant has captcha_on_signup persisted true", async () => {
    renderWithConfig(makeConfig({ captcha_on_signup: true }))

    // reset() runs in an effect, so the persisted value lands after the first paint.
    await waitFor(() => expect(captchaSwitch()).toBeChecked())
    // Interactive is the whole point: a tenant stuck in the outage has to be
    // able to turn it off from here.
    expect(captchaSwitch()).toBeEnabled()
  })

  it("warns in the description that registration is broken while the persisted flag is true", async () => {
    renderWithConfig(makeConfig({ captcha_on_signup: true }))

    await waitFor(() =>
      expect(screen.getByText(/registration will be rejected until this is turned off/i)).toBeInTheDocument(),
    )
  })

  it("lets an affected tenant turn the persisted flag off and save it", async () => {
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithConfig(makeConfig({ captcha_on_signup: true }))
    await waitFor(() => expect(captchaSwitch()).toBeChecked())

    await u().click(captchaSwitch())

    await waitFor(() => expect(captchaSwitch()).not.toBeChecked())
    // One-way by design: off is the safe state, and re-enabling it would break
    // signup again while no challenge widget exists.
    expect(captchaSwitch()).toBeDisabled()

    await u().click(screen.getByRole("button", { name: /save changes/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith(expect.objectContaining({ captcha_on_signup: false })),
    )
  })

  it("shows the switch OFF and locked when the tenant has captcha_on_signup persisted false", async () => {
    renderWithConfig(makeConfig({ captcha_on_signup: false }))

    await waitFor(() => expect(screen.getByRole("switch", { name: /self registration/i })).toBeChecked())
    expect(captchaSwitch()).not.toBeChecked()
    expect(captchaSwitch()).toBeDisabled()
  })

  it("does not let a tenant switch captcha on while the hosted form cannot answer the challenge", async () => {
    renderWithConfig(makeConfig({ captcha_on_signup: false }))
    await waitFor(() => expect(captchaSwitch()).toBeDisabled())

    await u().click(captchaSwitch())

    expect(captchaSwitch()).not.toBeChecked()
  })
})
