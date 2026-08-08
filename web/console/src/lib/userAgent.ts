// formatUserAgent produces a short, human-readable "Browser on OS" summary from
// a raw User-Agent string, for display only (device/session lists). It is
// intentionally dependency-free and best-effort — order matters because several
// UA strings embed each other (Edge contains "Chrome", Chrome contains "Safari").
export function formatUserAgent(ua?: string | null): string {
  const s = (ua ?? "").trim()
  if (!s) return "Unknown device"

  const browser = /Edg(?:e|A|iOS)?\//.test(s)
    ? "Edge"
    : /OPR\/|Opera/.test(s)
      ? "Opera"
      : /Firefox\/|FxiOS\//.test(s)
        ? "Firefox"
        : /Chrome\/|CriOS\//.test(s)
          ? "Chrome"
          : /Safari\//.test(s)
            ? "Safari"
            : ""

  const os = /Windows NT/.test(s)
    ? "Windows"
    : /iPhone|iPad|iPod/.test(s)
      ? "iOS"
      : /Mac OS X|Macintosh/.test(s)
        ? "macOS"
        : /Android/.test(s)
          ? "Android"
          : /Linux/.test(s)
            ? "Linux"
            : ""

  if (browser && os) return `${browser} on ${os}`
  if (browser) return browser
  if (os) return os
  // Never show nothing useful — fall back to a trimmed raw UA.
  return s.length > 40 ? `${s.slice(0, 40)}…` : s
}
