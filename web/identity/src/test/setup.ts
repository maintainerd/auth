import "@testing-library/jest-dom/vitest"
import { afterEach, vi } from "vitest"
import { cleanup } from "@testing-library/react"

afterEach(() => {
  cleanup()
})

// jsdom doesn't implement these APIs that Radix UI (popover/select/dropdown/dialog) relies on.
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as unknown as MediaQueryList
}

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
;(globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = ResizeObserverStub

Element.prototype.scrollIntoView = vi.fn()
;(Element.prototype as unknown as { hasPointerCapture: () => boolean }).hasPointerCapture = () => false
;(Element.prototype as unknown as { setPointerCapture: () => void }).setPointerCapture = () => {}
;(Element.prototype as unknown as { releasePointerCapture: () => void }).releasePointerCapture = () => {}
