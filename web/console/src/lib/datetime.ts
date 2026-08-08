/**
 * Boundary conversions between an <input type="datetime-local"> value and the
 * RFC3339 timestamps the API speaks.
 *
 * The two formats are not interchangeable: a datetime-local value is
 * "2026-08-05T14:30" — no seconds, no UTC offset — and the API parses schedule
 * fields with Go's time.RFC3339, rejecting both that shape and an empty string
 * (internal/tenant/validation_setting.go, optionalRFC3339Time). Submitting the
 * raw input value is therefore a guaranteed 422.
 */

/** Pad a date part to the two digits a datetime-local value requires. */
function pad(value: number): string {
  return String(value).padStart(2, "0")
}

/**
 * Converts a datetime-local input value into an RFC3339 timestamp, or null when
 * the field is empty.
 *
 * An offset-less date-time string is parsed as local time by JS, and
 * toISOString() renders it back as UTC, so the instant the user picked survives
 * the conversion regardless of the browser's timezone. Empty and unparseable
 * values become null rather than "": the API accepts JSON null as "nothing
 * scheduled" but 422s on an empty string.
 */
export function toRfc3339(value: string | null | undefined): string | null {
  if (value === null || value === undefined || value.trim() === "") return null
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return null
  return parsed.toISOString()
}

/**
 * Converts an RFC3339 timestamp from the API into the "YYYY-MM-DDTHH:mm" shape a
 * datetime-local input accepts, or null when nothing is scheduled.
 *
 * The parts are read with the local getters (not toISOString) so the wall clock
 * shown to the user is their own — rendering the UTC instant instead would shift
 * the visible time by the timezone offset and silently reschedule the window on
 * the next save.
 */
export function toDatetimeLocalInput(value: string | null | undefined): string | null {
  if (value === null || value === undefined || value.trim() === "") return null
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return null
  return (
    `${parsed.getFullYear()}-${pad(parsed.getMonth() + 1)}-${pad(parsed.getDate())}` +
    `T${pad(parsed.getHours())}:${pad(parsed.getMinutes())}`
  )
}
