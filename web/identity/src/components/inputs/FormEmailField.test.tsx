import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { FormCodeField, FormEmailField, FormPasswordFieldWithPolicy, FormPhoneField, FormUrlField } from './index'

describe('identity input wrappers', () => {
  it('centralizes specialized input attributes on reusable fields', () => {
    render(
      <div>
        <FormEmailField label="Email" />
        <FormPhoneField label="Phone" />
        <FormUrlField label="Avatar URL" />
        <FormCodeField label="Verification code" numeric />
        <FormPasswordFieldWithPolicy label="Password" />
      </div>,
    )

    expect(screen.getByLabelText('Email')).toHaveAttribute('type', 'email')
    expect(screen.getByLabelText('Phone')).toHaveAttribute('type', 'tel')
    expect(screen.getByLabelText('Avatar URL')).toHaveAttribute('type', 'url')
    expect(screen.getByLabelText('Verification code')).toHaveAttribute('inputmode', 'numeric')
    expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'password')
  })
})
