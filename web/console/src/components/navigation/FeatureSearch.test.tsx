import { describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import { FeatureSearch } from "./FeatureSearch"

const navigateMock = vi.fn()
vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return { ...actual, useNavigate: () => navigateMock }
})

function renderSearch() {
  render(
    <MemoryRouter initialEntries={["/dashboard"]}>
      <FeatureSearch />
    </MemoryRouter>,
  )
}

describe("FeatureSearch", () => {
  it("shows a type-to-search hint when the palette is empty", async () => {
    const user = userEvent.setup()
    renderSearch()
    await user.click(screen.getByLabelText("Search features"))
    expect(await screen.findByText("Type to search features.")).toBeInTheDocument()
  })

  it("shows the best-matching result first when typing", async () => {
    const user = userEvent.setup()
    renderSearch()
    await user.click(screen.getByLabelText("Search features"))
    const input = await screen.findByPlaceholderText("Search features...")
    await user.type(input, "email")

    const items = screen.getAllByRole("option")
    expect(items[0]).toHaveTextContent("Email Configuration")
  })

  it("filters by keyword", async () => {
    const user = userEvent.setup()
    renderSearch()
    await user.click(screen.getByLabelText("Search features"))
    const input = await screen.findByPlaceholderText("Search features...")
    await user.type(input, "saml")
    expect(screen.getByText("Identity Providers")).toBeInTheDocument()
  })

  it("navigates on select", async () => {
    const user = userEvent.setup()
    renderSearch()
    await user.click(screen.getByLabelText("Search features"))
    const input = await screen.findByPlaceholderText("Search features...")
    await user.type(input, "mfa")
    await user.click(screen.getByText("MFA Configuration"))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/security/mfa/configure"))
  })

  it("shows an empty state when nothing matches", async () => {
    const user = userEvent.setup()
    renderSearch()
    await user.click(screen.getByLabelText("Search features"))
    const input = await screen.findByPlaceholderText("Search features...")
    await user.type(input, "zzzz")
    expect(await screen.findByText("No feature found.")).toBeInTheDocument()
  })
})
