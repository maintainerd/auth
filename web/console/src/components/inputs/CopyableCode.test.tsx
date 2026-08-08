import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { CopyableCode } from "./CopyableCode"

const showSuccessMock = vi.fn()
const showErrorMock = vi.fn()

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("CopyableCode", () => {
  beforeEach(() => vi.clearAllMocks())

  it("renders the value and copies it verbatim", async () => {
    renderWithProviders(<CopyableCode value="partner-signup" label="Flow name" />)

    expect(screen.getByText("partner-signup")).toBeInTheDocument()
    await u().click(screen.getByRole("button", { name: /copy flow name/i }))

    await waitFor(() => expect(showSuccessMock).toHaveBeenCalledWith("Flow name copied to clipboard"))
    await expect(navigator.clipboard.readText()).resolves.toBe("partner-signup")
  })

  // The label drives both the accessible action and the toast, so they cannot
  // drift apart the way hand-rolled copies did.
  it("derives the accessible name and the toast from one label", async () => {
    renderWithProviders(<CopyableCode value="https://x/register" label="Registration link" />)

    await u().click(screen.getByRole("button", { name: /copy registration link/i }))
    await waitFor(() =>
      expect(showSuccessMock).toHaveBeenCalledWith("Registration link copied to clipboard"),
    )
  })

  it("surfaces a clipboard failure instead of appearing to succeed", async () => {
    const denied = new Error("permission denied")
    vi.spyOn(navigator.clipboard, "writeText").mockRejectedValueOnce(denied)

    renderWithProviders(<CopyableCode value="v" label="Value" />)
    await u().click(screen.getByRole("button", { name: /copy value/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(denied))
    expect(showSuccessMock).not.toHaveBeenCalled()
  })

  // Table rows navigate on click; copying must not also trigger that.
  it("stops propagation only when asked", async () => {
    const onRowClick = vi.fn()

    const { unmount } = renderWithProviders(
      <div onClick={onRowClick}>
        <CopyableCode value="v" label="Value" stopPropagation />
      </div>,
    )
    await u().click(screen.getByRole("button", { name: /copy value/i }))
    expect(onRowClick).not.toHaveBeenCalled()
    unmount()

    renderWithProviders(
      <div onClick={onRowClick}>
        <CopyableCode value="v" label="Value" />
      </div>,
    )
    await u().click(screen.getByRole("button", { name: /copy value/i }))
    expect(onRowClick).toHaveBeenCalled()
  })

  it("is a button, so it never submits a surrounding form", () => {
    renderWithProviders(<CopyableCode value="v" label="Value" />)
    expect(screen.getByRole("button", { name: /copy value/i })).toHaveAttribute("type", "button")
  })
})
