/**
 * Client-side avatar compression.
 *
 * The server caps uploads at 2 MB, and a phone camera photo is routinely 4–8 MB
 * — so without this, the common case is a rejection the user cannot act on
 * ("use a smaller image" is not something most people can do). Compressing here
 * turns that into a silent success, and it also saves the upload bandwidth.
 *
 * An avatar is displayed at ~64–256 CSS pixels, so the pixel budget below is
 * generous even for a retina display. Nearly all of the size reduction comes
 * from the downscale rather than the quality search.
 */

/** Longest edge kept after downscaling. */
const MAX_DIMENSION = 512

/** Quality steps tried, highest first. */
const QUALITY_STEPS = [0.92, 0.85, 0.75, 0.65, 0.5]

export interface CompressResult {
  file: File
  /** True when the image was re-encoded rather than passed through unchanged. */
  compressed: boolean
  originalBytes: number
}

/**
 * Returns a file guaranteed under maxBytes where possible.
 *
 * Passes the original through untouched when it already fits AND is within the
 * pixel budget — re-encoding a small image only loses quality for no gain.
 *
 * Animated GIFs are passed through regardless: a canvas re-encode captures a
 * single frame, so compressing one silently turns an animation into a still.
 * An oversized GIF is better refused by the server than silently broken here.
 */
export async function compressImage(file: File, maxBytes: number): Promise<CompressResult> {
  const originalBytes = file.size

  if (file.type === "image/gif") {
    return { file, compressed: false, originalBytes }
  }

  const bitmap = await loadBitmap(file)
  try {
    const scale = Math.min(1, MAX_DIMENSION / Math.max(bitmap.width, bitmap.height))
    if (file.size <= maxBytes && scale === 1) {
      return { file, compressed: false, originalBytes }
    }

    const width = Math.max(1, Math.round(bitmap.width * scale))
    const height = Math.max(1, Math.round(bitmap.height * scale))

    const canvas = document.createElement("canvas")
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext("2d")
    if (!ctx) return { file, compressed: false, originalBytes }
    ctx.drawImage(bitmap, 0, 0, width, height)

    // WebP preserves transparency and beats JPEG at the same quality. Fall back
    // to JPEG where it is unsupported; a PNG re-encode would often be LARGER
    // than the original for a photograph, which defeats the point.
    const type = canvas.toDataURL("image/webp").startsWith("data:image/webp") ? "image/webp" : "image/jpeg"

    for (const quality of QUALITY_STEPS) {
      const blob = await toBlob(canvas, type, quality)
      if (blob && blob.size <= maxBytes) {
        return {
          file: new File([blob], renameFor(file.name, type), { type }),
          compressed: true,
          originalBytes,
        }
      }
    }

    // Even the lowest quality did not fit. Hand back the smallest attempt and
    // let the server refuse it, rather than silently uploading the original.
    const smallest = await toBlob(canvas, type, QUALITY_STEPS[QUALITY_STEPS.length - 1])
    if (smallest) {
      return {
        file: new File([smallest], renameFor(file.name, type), { type }),
        compressed: true,
        originalBytes,
      }
    }
    return { file, compressed: false, originalBytes }
  } finally {
    bitmap.close?.()
  }
}

/**
 * Decodes to a bitmap, preferring createImageBitmap because it decodes off the
 * main thread. Falls back to an <img> for browsers without it.
 */
async function loadBitmap(file: File): Promise<ImageBitmap> {
  if (typeof createImageBitmap === "function") {
    return createImageBitmap(file)
  }
  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image()
      el.onload = () => resolve(el)
      el.onerror = () => reject(new Error("The image could not be read"))
      el.src = url
    })
    // Shaped like an ImageBitmap so the caller needs no branch; close() is
    // optional on the type and absent here.
    return img as unknown as ImageBitmap
  } finally {
    URL.revokeObjectURL(url)
  }
}

function toBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}

function renameFor(name: string, type: string): string {
  const base = name.replace(/\.[^./\\]+$/, "") || "avatar"
  return `${base}.${type === "image/webp" ? "webp" : "jpg"}`
}
