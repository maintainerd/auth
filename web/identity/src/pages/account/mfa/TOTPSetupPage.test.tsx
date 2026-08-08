import type { ReactNode } from "react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import TOTPSetupPage from "./TOTPSetupPage"

const { beginMock, finishMock, statusMock, toCanvasMock } = vi.hoisted(() => ({
  beginMock: vi.fn(),
  finishMock: vi.fn(),
  statusMock: vi.fn(),
  toCanvasMock: vi.fn(),
}))

vi.mock("@/components/layout/AccountLayout", () => ({
  default: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

vi.mock("qrcode", () => ({
  default: { toCanvas: (...args: unknown[]) => toCanvasMock(...args) },
}))

vi.mock("@/services/api/mfa", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/api/mfa")>()),
  fetchMFAStatus: () => statusMock(),
  beginTOTPEnrollment: () => beginMock(),
  finishTOTPEnrollment: (v: unknown) => finishMock(v),
}))

const OTPAUTH_URI = "otpauth://totp/Maintainerd:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Maintainerd"

// `qr_code_url` is an otpauth:// DEEP LINK, not an image URL. It was being fed
// straight to <img src>, which a browser cannot load — the page rendered a
// broken image and manual key entry was the only way to enrol.
describe("TOTPSetupPage — QR rendering", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    toCanvasMock.mockImplementation((_canvas, _url, _opts, cb) => cb?.(null))
    statusMock.mockResolvedValue({
      is_totp_enabled: false,
      is_webauthn_enabled: false,
      is_sms_available: false,
      is_email_otp_available: false,
      backup_codes_count: 0,
      webauthn_keys: [],
      allowed_methods: ["totp"],
      mfa_required: false,
    })
    beginMock.mockResolvedValue({ secret: "JBSWY3DPEHPK3PXP", qr_code_url: OTPAUTH_URI })
    finishMock.mockResolvedValue({ codes: ["AAAA-1111", "BBBB-2222"] })
  })

  it("draws the QR from the otpauth URI instead of loading it as an image", async () => {
    renderWithProviders(<TOTPSetupPage />)

    const start = await screen.findByRole("button", { name: /begin setup/i })
    await userEvent.setup({ pointerEventsCheck: 0 }).click(start)

    await waitFor(() => expect(toCanvasMock).toHaveBeenCalled())
    expect(toCanvasMock.mock.calls[0][1]).toBe(OTPAUTH_URI)

    // An <img> pointing at otpauth:// is the bug; it must not come back.
    const broken = Array.from(document.querySelectorAll("img")).some(
      (img) => (img as HTMLImageElement).getAttribute("src")?.startsWith("otpauth://"),
    )
    expect(broken).toBe(false)
  })

  // The console gates codes behind a reveal and offers Copy AND Download. The
  // identity app showed them immediately with copy only, so a user with no
  // clipboard access (or who wanted a file) had no way to save them.
  it("gates backup codes behind a reveal, then offers copy and download", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<TOTPSetupPage />)

    await user.click(await screen.findByRole("button", { name: /begin setup/i }))
    await user.click(await screen.findByRole("button", { name: /i've scanned the code/i }))
    await user.type(await screen.findByLabelText(/verification code/i), "123456")
    await user.click(screen.getByRole("button", { name: /verify & enable/i }))

    // Hidden until deliberately revealed — the dialog may be on a shared screen
    // and the codes cannot be shown again later.
    expect(await screen.findByRole("button", { name: /reveal codes/i })).toBeInTheDocument()
    expect(screen.queryByText("AAAA-1111")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /download/i })).not.toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /reveal codes/i }))

    expect(screen.getByText("AAAA-1111")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /download/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /copy all/i })).toBeInTheDocument()
  })

  // A blank square tells the user nothing. The manual key still works, so say so.
  it("falls back to a readable message when the QR cannot be drawn", async () => {
    toCanvasMock.mockImplementation((_canvas, _url, _opts, cb) => cb?.(new Error("boom")))
    renderWithProviders(<TOTPSetupPage />)

    const start = await screen.findByRole("button", { name: /begin setup/i })
    await userEvent.setup({ pointerEventsCheck: 0 }).click(start)

    expect(await screen.findByText(/could not draw the qr code/i)).toBeInTheDocument()
  })
})
