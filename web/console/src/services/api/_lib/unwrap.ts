/**
 * Response unwrapping helpers
 *
 * Every backend response is an `ApiResponse<T>` envelope (`{ success, data, message }`).
 * These helpers centralize the "return data or throw" logic that was previously
 * copy-pasted into every service function.
 */

import { ApiError } from '../client'
import type { ApiResponse } from '../types'

/**
 * Returns `res.data` when the call succeeded, otherwise throws an ApiError.
 * Uses `!= null` so legitimately falsy payloads (empty array, 0, false, "") are returned.
 */
export function unwrap<T>(res: ApiResponse<T>, action: string): T {
  if (res.success && res.data != null) {
    return res.data
  }
  throw new ApiError({
    message: res.message || `Failed to ${action}`,
    status: 0,
  })
}

/**
 * Asserts a successful response for calls with no meaningful body (e.g. delete).
 */
export function assertSuccess(res: ApiResponse<unknown>, action: string): void {
  if (!res.success) {
    throw new ApiError({
      message: res.message || `Failed to ${action}`,
      status: 0,
    })
  }
}
