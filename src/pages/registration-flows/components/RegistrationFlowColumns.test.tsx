import { describe, it, expect, vi } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { registrationFlowColumns } from "./RegistrationFlowColumns"
import { DataTable } from "@/components/data-table"
import type { RegistrationFlow } from "@/services/api/registration-flows/types"

const { showSuccessMock } = vi.hoisted(() => ({
  showSuccessMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useDeleteRegistrationFlow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateRegistrationFlowStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: vi.fn() }),
}))

function makeFlow(overrides: Partial<RegistrationFlow> = {}): RegistrationFlow {
  return {
    registration_flow_id: "f1",
    name: "seller-signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "c-uuid",
    verification_required: true,
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: RegistrationFlow[] }) {
  const table = useReactTable({
    data,
    columns: registrationFlowColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={registrationFlowColumns.length} />
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("registrationFlowColumns", () => {
  it("renders all cell branches across multiple flows", () => {
    const data = [
      makeFlow({ registration_flow_id: "f1", name: "seller-signup", status: "active", is_system: true }),
      makeFlow({ registration_flow_id: "f2", name: "buyer-signup", status: "inactive", is_system: false }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "Registration Flow" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // Names + descriptions. The name is the public link selector, so it is the
    // copyable code cell.
    expect(screen.getByText("seller-signup")).toBeInTheDocument()
    expect(screen.getByText("buyer-signup")).toBeInTheDocument()
    expect(screen.getAllByText("Sellers onboard here").length).toBe(2)

    // Status badges
    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("active")).toBeInTheDocument()
    expect(within(tbody).getByText("inactive")).toBeInTheDocument()

    // System badge only for the system flow
    expect(screen.getAllByText("System").length).toBe(1)

    // Actions menu trigger per row
    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })

  it("copies the flow name from the listing", async () => {
    // user-event installs its own clipboard stub, so read the copied value back
    // through it and assert the observable toast.
    const user = u()
    renderWithProviders(<Harness data={[makeFlow()]} />)

    await user.click(screen.getByRole("button", { name: /copy flow name/i }))

    await waitFor(() =>
      expect(showSuccessMock).toHaveBeenCalledWith("Flow name copied to clipboard"),
    )
    await expect(navigator.clipboard.readText()).resolves.toBe("seller-signup")
  })

  it("renders a status badge for every row (no blank status cell)", () => {
    renderWithProviders(<Harness data={[makeFlow({ status: "inactive" })]} />)
    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("inactive")).toBeInTheDocument()
  })
})
