import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { UserMetadata } from "./UserMetadata"
import type { User, UserMetadata as UserMetadataType } from "@/services/api/users/types"

function makeUser(metadata: UserMetadataType | undefined): User {
  return {
    user_id: "u1",
    username: "jdoe",
    fullname: "John Doe",
    email: "jdoe@example.com",
    phone: "123",
    is_email_verified: true,
    is_phone_verified: false,
    is_profile_completed: true,
    is_account_completed: false,
    status: "active",
    metadata: metadata as UserMetadataType,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  }
}

describe("UserMetadata", () => {
  it("renders the empty state when there is no metadata", () => {
    renderWithProviders(<UserMetadata user={makeUser({})} />)
    expect(screen.getByText("No metadata")).toBeInTheDocument()
  })

  it("treats null metadata as empty", () => {
    renderWithProviders(<UserMetadata user={makeUser(undefined)} />)
    expect(screen.getByText("No metadata")).toBeInTheDocument()
  })

  it("formats each value type and renders rows", () => {
    const metadata = {
      str: "hello",
      num: 42,
      boolTrue: true,
      boolFalse: false,
      arr: ["a", "b"],
      obj: { nested: 1 },
      nullish: null,
    } as unknown as UserMetadataType
    renderWithProviders(<UserMetadata user={makeUser(metadata)} />)

    expect(screen.getByText("str")).toBeInTheDocument()
    expect(screen.getByText("hello")).toBeInTheDocument()
    expect(screen.getByText("42")).toBeInTheDocument()
    expect(screen.getByText("true")).toBeInTheDocument()
    expect(screen.getByText("false")).toBeInTheDocument()
    expect(screen.getByText("a, b")).toBeInTheDocument()
    expect(screen.getByText('{"nested":1}')).toBeInTheDocument()
    // null/undefined render as the em-dash placeholder.
    expect(screen.getByText("—")).toBeInTheDocument()
  })
})
