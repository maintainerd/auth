/**
 * API Types
 * Shared API type definitions used across multiple services.
 *
 * Per-resource types are co-located in `services/api/<resource>/types.ts`.
 */

/**
 * Generic API response envelope returned by every backend endpoint.
 *
 * @template T - The type of the data payload (optional)
 *
 * @example
 * // Success response with data
 * ApiResponse<{ user_id: string }>
 *
 * @example
 * // Success response without data (success/message only)
 * ApiResponse
 *
 * @example
 * // Error response
 * { success: false, error: "Invalid credentials", details: "Username not found" }
 */
export interface ApiResponse<T = undefined> {
  success: boolean
  data?: T
  message?: string
  error?: string | object
  details?: string | object
}

/**
 * Paginated list payload returned inside `ApiResponse<PaginatedResponse<T>>`.
 *
 * @template T - The type of an individual row.
 */
export interface PaginatedResponse<T> {
  rows: T[]
  total: number
  page: number
  limit: number
  total_pages: number
}
