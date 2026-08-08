import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { FormScopeField } from './FormScopeField'

describe('FormScopeField', () => {
  it('renders the comma-separated value', () => {
    render(
      <FormScopeField label="Scopes" value="openid, profile" onValueChange={() => {}} />,
    )
    expect(screen.getByDisplayValue('openid, profile')).toBeInTheDocument()
  })

  it('surfaces a validation error for a malformed token', () => {
    render(
      <FormScopeField label="Scopes" value="openid, bad scope!" onValueChange={() => {}} />,
    )
    expect(screen.getByText(/invalid scope/i)).toBeInTheDocument()
  })

  it('propagates edits from the text input', () => {
    const onValueChange = vi.fn()
    render(<FormScopeField label="Scopes" value="openid" onValueChange={onValueChange} />)
    fireEvent.change(screen.getByDisplayValue('openid'), { target: { value: 'openid, email' } })
    expect(onValueChange).toHaveBeenCalledWith('openid, email')
  })
})
