import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { compressImage } from "./imageCompression"

const MAX = 2 * 1024 * 1024

function fakeFile(name: string, type: string, size: number): File {
  const f = new File([new Uint8Array(1)], name, { type })
  // jsdom will not synthesise a large body; size is what the logic branches on.
  Object.defineProperty(f, "size", { value: size })
  return f
}

beforeEach(() => {
  vi.stubGlobal("createImageBitmap", vi.fn(async () => ({ width: 4000, height: 3000, close: vi.fn() })))
  // jsdom has no canvas encoder; stand in for one that honours the quality arg.
  vi.spyOn(HTMLCanvasElement.prototype, "toDataURL").mockReturnValue("data:image/webp;base64,AA")
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({ drawImage: vi.fn() } as never)
  vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(function (
    this: HTMLCanvasElement,
    cb: BlobCallback,
    type?: string,
    quality?: number,
  ) {
    const bytes = Math.round(3_000_000 * (quality ?? 1))
    cb(Object.defineProperty(new Blob(["x"], { type }), "size", { value: bytes }))
  } as never)
})

// These stub globals and patch canvas prototypes; without restoring them the
// next test file in the same worker inherits a fake canvas.
afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

// A phone photo is routinely 4–8 MB. Rejecting it outright asks the user to do
// something most people cannot, so the client shrinks it first.
describe("compressImage", () => {
  it("compresses an oversized image under the limit", async () => {
    const result = await compressImage(fakeFile("photo.jpg", "image/jpeg", 6_000_000), MAX)
    expect(result.compressed).toBe(true)
    expect(result.file.size).toBeLessThanOrEqual(MAX)
    expect(result.originalBytes).toBe(6_000_000)
  })

  // Re-encoding a small image only loses quality for no gain.
  it("passes through an image already within budget", async () => {
    vi.stubGlobal("createImageBitmap", vi.fn(async () => ({ width: 256, height: 256, close: vi.fn() })))
    const original = fakeFile("small.png", "image/png", 40_000)
    const result = await compressImage(original, MAX)
    expect(result.compressed).toBe(false)
    expect(result.file).toBe(original)
  })

  // A canvas re-encode keeps one frame, so compressing an animation silently
  // turns it into a still. Better to pass it through and let the server decide.
  it("never re-encodes a GIF", async () => {
    const gif = fakeFile("loop.gif", "image/gif", 5_000_000)
    const result = await compressImage(gif, MAX)
    expect(result.compressed).toBe(false)
    expect(result.file).toBe(gif)
  })

  // Downscaling alone is most of the win: an avatar renders at ~64–256px.
  it("downscales an oversized canvas even when the file is small", async () => {
    const result = await compressImage(fakeFile("huge.png", "image/png", 100_000), MAX)
    expect(result.compressed).toBe(true)
  })
})
