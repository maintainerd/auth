/**
 * Debounced Search Hook
 * Custom hook for debouncing search input with Enter key support
 */

import { useState, useEffect, useCallback, useRef } from 'react'

export interface UseDebouncedSearchOptions {
  /**
   * Initial search value
   */
  initialValue?: string
  /**
   * Debounce delay in milliseconds
   * @default 500
   */
  delay?: number
  /**
   * Callback when debounced value changes
   */
  onDebouncedChange?: (value: string) => void
}

export interface UseDebouncedSearchReturn {
  /**
   * Current input value (immediate, not debounced)
   */
  searchInput: string
  /**
   * Debounced search value
   */
  debouncedValue: string
  /**
   * Handler for input change events
   */
  handleSearchChange: (value: string) => void
  /**
   * Handler for keyboard events (Enter key triggers immediate search)
   */
  handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void
  /**
   * Manually set the search value (bypasses debounce)
   */
  setSearchValue: (value: string) => void
}

/**
 * Hook for debounced search input with Enter key support
 * 
 * @example
 * ```tsx
 * const { searchInput, handleSearchChange, handleKeyDown } = useDebouncedSearch({
 *   initialValue: '',
 *   delay: 500,
 *   onDebouncedChange: (value) => setFilter(value)
 * })
 * 
 * return (
 *   <Input
 *     value={searchInput}
 *     onChange={(e) => handleSearchChange(e.target.value)}
 *     onKeyDown={handleKeyDown}
 *   />
 * )
 * ```
 */
export function useDebouncedSearch({
  initialValue = '',
  delay = 500,
  onDebouncedChange
}: UseDebouncedSearchOptions = {}): UseDebouncedSearchReturn {
  const [searchInput, setSearchInput] = useState(initialValue)
  const [debouncedValue, setDebouncedValue] = useState(initialValue)
  const debounceTimerRef = useRef<number | null>(null)

  // Sync with external changes to initial value
  useEffect(() => {
    setSearchInput(initialValue)
    setDebouncedValue(initialValue)
  }, [initialValue])

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current)
      }
    }
  }, [])

  // Notify parent when debounced value changes
  useEffect(() => {
    if (onDebouncedChange) {
      onDebouncedChange(debouncedValue)
    }
  }, [debouncedValue, onDebouncedChange])

  // Debounced search handler
  const handleSearchChange = useCallback((value: string) => {
    setSearchInput(value)
    
    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
    }
    
    // Set new timer for debounced update
    debounceTimerRef.current = setTimeout(() => {
      setDebouncedValue(value)
    }, delay) as unknown as number
  }, [delay])

  // Handle Enter key press for immediate search
  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      // Clear debounce timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current)
      }
      // Trigger immediate search
      setDebouncedValue(searchInput)
    }
  }, [searchInput])

  // Manually set search value (bypasses debounce)
  const setSearchValue = useCallback((value: string) => {
    setSearchInput(value)
    setDebouncedValue(value)
    
    // Clear any pending debounce
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
    }
  }, [])

  return {
    searchInput,
    debouncedValue,
    handleSearchChange,
    handleKeyDown,
    setSearchValue
  }
}

