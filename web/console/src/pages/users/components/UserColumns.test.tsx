import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { userColumns } from "./UserColumns"
import { DataTable } from "@/components/data-table"
import type { User, UserStatus } from "@/services/api/users/types"

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useUsers", () => ({
  useUpdateUserStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteUser: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeUser(overrides: Partial<User> = {}): User {
  return {
    user_id: "u1",
    username: "jdoe",
    fullname: "John Doe",
    email: "jdoe@example.com",
    phone: "12345",
    is_email_verified: true,
    is_phone_verified: false,
    is_profile_completed: true,
    is_account_completed: false,
    status: "active",
    metadata: {},
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: User[] }) {
  const table = useReactTable({
    data,
    columns: userColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={userColumns.length} />
}

const statuses: UserStatus[] = ["active", "inactive", "pending", "suspended"]

describe("userColumns", () => {
  it("renders all cell branches across multiple users", () => {
    const data: User[] = [
      makeUser({
        user_id: "u1",
        username: "active_user",
        status: "active",
      }),
      makeUser({
        user_id: "u2",
        username: "inactive_user",
        status: "inactive",
      }),
      makeUser({ user_id: "u3", username: "pending_user", status: "pending" }),
      makeUser({ user_id: "u4", username: "suspended_user", status: "suspended" }),
      makeUser({ user_id: "u5", username: "no_name_user", fullname: "", status: "active" }),
      makeUser({ user_id: "u6", username: " ", fullname: " ", status: "active" }),
    ]

    renderWithProviders(<Harness data={data} />)

    expect(screen.getByRole("button", { name: "Username" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    for (const s of statuses) {
      expect(screen.getAllByText(s).length).toBeGreaterThanOrEqual(1)
    }

    expect(screen.getByText("active_user")).toBeInTheDocument()
    expect(screen.getByText("inactive_user")).toBeInTheDocument()
    // The username shows once in its column (the avatar shows its initials).
    expect(screen.getByText("no_name_user")).toBeInTheDocument()
    // The blank-name/blank-username user still shows the "?" avatar fallback.
    expect(screen.getByText("?")).toBeInTheDocument()

    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })
})
