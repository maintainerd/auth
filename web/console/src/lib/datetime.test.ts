import { describe, expect, it } from "vitest"

import { toDatetimeLocalInput, toRfc3339 } from "./datetime"

// The shape Go's time.RFC3339 accepts. Kept here so a regression that emits a
// bare datetime-local value fails the assertion rather than the API.
const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/

describe("toRfc3339", () => {
  it("converts a datetime-local value to an RFC3339 timestamp at the same instant", () => {
    const result = toRfc3339("2026-08-05T14:30")

    expect(result).toMatch(RFC3339)
    expect(new Date(result as string).getTime()).toBe(new Date(2026, 7, 5, 14, 30, 0, 0).getTime())
  })

  it("passes an already-RFC3339 value through without moving the instant", () => {
    const result = toRfc3339("2026-08-05T14:30:00Z")

    expect(result).toMatch(RFC3339)
    expect(new Date(result as string).getTime()).toBe(new Date("2026-08-05T14:30:00Z").getTime())
  })

  it.each([
    ["an empty string", ""],
    ["whitespace", "   "],
    ["null", null],
    ["undefined", undefined],
  ])("returns null for %s, because the API rejects an empty string but accepts null", (_label, value) => {
    expect(toRfc3339(value)).toBeNull()
  })

  it("returns null for an unparseable value instead of forwarding a 422", () => {
    expect(toRfc3339("not-a-date")).toBeNull()
  })
})

describe("toDatetimeLocalInput", () => {
  it("renders a stored timestamp as the local wall clock the input expects", () => {
    const stored = new Date(2026, 7, 5, 14, 30).toISOString()

    expect(toDatetimeLocalInput(stored)).toBe("2026-08-05T14:30")
  })

  it("zero-pads month, day, hour and minute so the input can parse the value", () => {
    const stored = new Date(2026, 0, 2, 3, 4).toISOString()

    expect(toDatetimeLocalInput(stored)).toBe("2026-01-02T03:04")
  })

  it.each([
    ["an empty string", ""],
    ["whitespace", "   "],
    ["null", null],
    ["undefined", undefined],
  ])("returns null for %s", (_label, value) => {
    expect(toDatetimeLocalInput(value)).toBeNull()
  })

  it("returns null for an unparseable value rather than showing 'Invalid Date'", () => {
    expect(toDatetimeLocalInput("not-a-date")).toBeNull()
  })
})

describe("round trip", () => {
  it("keeps the wall clock the user typed stable across a save", () => {
    const typed = "2026-08-05T14:30"

    expect(toDatetimeLocalInput(toRfc3339(typed))).toBe(typed)
  })

  it("keeps the instant stored by the API stable across a load and re-save", () => {
    const stored = "2026-08-05T14:30:00Z"

    const reSaved = toRfc3339(toDatetimeLocalInput(stored))

    expect(reSaved).toMatch(RFC3339)
    expect(new Date(reSaved as string).getTime()).toBe(new Date(stored).getTime())
  })

  it("keeps an offset timestamp's instant stable, whatever the local timezone is", () => {
    const stored = "2026-08-05T14:30:00+08:00"

    const reSaved = toRfc3339(toDatetimeLocalInput(stored))

    expect(new Date(reSaved as string).getTime()).toBe(new Date(stored).getTime())
  })

  it("maps a cleared field to null in both directions", () => {
    expect(toRfc3339(toDatetimeLocalInput(null))).toBeNull()
    expect(toDatetimeLocalInput(toRfc3339(""))).toBeNull()
  })
})
