import { format } from "date-fns"

// safeFormat formats a date value with date-fns but never throws on a missing or
// malformed timestamp — it returns an em dash instead, so one bad value can't
// crash a whole list/tab render.
export function safeFormat(
  value: string | number | Date | null | undefined,
  pattern: string,
): string {
  if (value === null || value === undefined || value === "") return "—"
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return "—"
  return format(d, pattern)
}
