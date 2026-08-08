/**
 * Custom hook for toast notifications
 * Provides consistent toast messaging across the application
 */

import { useCallback } from "react"
import { toast } from "react-toastify"

export interface UseToastOptions {
  defaultErrorTitle?: string
  defaultErrorDescription?: string
}

export interface ParsedError {
  message: string
  fieldErrors?: Record<string, string>
  isValidationError: boolean
}

export function useToast(options: UseToastOptions = {}) {
  const {
    defaultErrorDescription = "An unexpected error occurred"
  } = options

  const showError = useCallback((
    error: unknown,
    _title?: string,
    description?: string
  ) => {
    let errorMessage = description || defaultErrorDescription

    // Extract error message from different error types
    if (error instanceof Error) {
      errorMessage = error.message

      // Check if this is an API error with responseData
      const apiError = error as { responseData?: { error: string; details?: unknown } }
      if (apiError.responseData) {
        const { error: apiErrorMessage, details } = apiError.responseData

        if (details) {
          if (typeof details === 'string') {
            // Simple string details
            errorMessage = `${apiErrorMessage}: ${details}`
          } else if (typeof details === 'object' && details !== null) {
            // Object with field-specific errors
            const fieldErrors = Object.entries(details)
              .map(([field, message]) => `${field}: ${message}`)
              .join(', ')
            errorMessage = `${apiErrorMessage} - ${fieldErrors}`
          }
        } else {
          errorMessage = apiErrorMessage
        }
      }
    } else if (typeof error === 'string') {
      errorMessage = error
    } else if (error && typeof error === 'object' && 'message' in error) {
      errorMessage = String(error.message)
    }

    toast.error(errorMessage)
  }, [defaultErrorDescription])

  const showSuccess = useCallback((
    title: string,
    description?: string
  ) => {
    const message = description ? `${title}: ${description}` : title
    toast.success(message)
  }, [])

  const showInfo = useCallback((
    title: string,
    description?: string
  ) => {
    const message = description ? `${title}: ${description}` : title
    toast.info(message)
  }, [])

  const showWarning = useCallback((
    title: string,
    description?: string
  ) => {
    const message = description ? `${title}: ${description}` : title
    toast.warning(message)
  }, [])

  const parseError = useCallback((error: unknown): ParsedError => {
    let message = defaultErrorDescription
    let fieldErrors: Record<string, string> | undefined
    let isValidationError = false

    if (error instanceof Error) {
      message = error.message

      // Check if this is an API error with responseData
      const apiError = error as { responseData?: { error: string; details?: unknown } }
      if (apiError.responseData) {
        const { error: apiErrorMessage, details } = apiError.responseData

        if (details) {
          if (typeof details === 'string') {
            // Simple string details
            message = `${apiErrorMessage}: ${details}`
          } else if (typeof details === 'object' && details !== null) {
            // Object with field-specific errors - this is a validation error
            fieldErrors = details as Record<string, string>
            message = apiErrorMessage
            isValidationError = true
          }
        } else {
          message = apiErrorMessage
        }
      }
    } else if (typeof error === 'string') {
      message = error
    } else if (error && typeof error === 'object' && 'message' in error) {
      message = String(error.message)
    }

    return {
      message,
      fieldErrors,
      isValidationError
    }
  }, [defaultErrorDescription])

  return {
    showError,
    showSuccess,
    showInfo,
    showWarning,
    parseError
  }
}
