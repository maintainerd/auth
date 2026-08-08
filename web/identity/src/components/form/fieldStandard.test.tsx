/**
 * One standard for every field.
 *
 * These tests exist because the field components had drifted into "each input
 * its own universe": three different error colours, label→input gaps of 12px,
 * 20px and 6px, and aria wiring on some controls but not others. Each case
 * below asserts a property of the shared FieldShell against EVERY field
 * component, so a new field that hand-rolls its own scaffolding fails here
 * rather than shipping a fourth variant.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { FormInputField } from './FormInputField'
import { FormPasswordField } from './FormPasswordField'
import { FormSelectField } from './FormSelectField'
import { FormConsentCheckbox } from './FormConsentCheckbox'
import { FormEmailField } from '@/components/inputs/FormEmailField'
import { FormPhoneField } from '@/components/inputs/FormPhoneField'
import { FormUrlField } from '@/components/inputs/FormUrlField'
import { FormCodeField } from '@/components/inputs/FormCodeField'
import { FormPasswordFieldWithPolicy } from '@/components/inputs/FormPasswordFieldWithPolicy'

/** Every label+control component, rendered the way callers use them. */
const FIELDS = [
  { name: 'FormInputField', render: (p: Record<string, unknown>) => <FormInputField label="Name" {...p} /> },
  { name: 'FormPasswordField', render: (p: Record<string, unknown>) => <FormPasswordField label="Password" {...p} /> },
  { name: 'FormEmailField', render: (p: Record<string, unknown>) => <FormEmailField label="Email" {...p} /> },
  { name: 'FormPhoneField', render: (p: Record<string, unknown>) => <FormPhoneField label="Phone" {...p} /> },
  { name: 'FormUrlField', render: (p: Record<string, unknown>) => <FormUrlField label="Website" {...p} /> },
  { name: 'FormCodeField', render: (p: Record<string, unknown>) => <FormCodeField label="Code" {...p} /> },
  {
    name: 'FormPasswordFieldWithPolicy',
    render: (p: Record<string, unknown>) => <FormPasswordFieldWithPolicy label="Password" {...p} />,
  },
  {
    name: 'FormSelectField',
    render: (p: Record<string, unknown>) => <FormSelectField label="Role" options={[{ value: 'a', label: 'A' }]} {...p} />,
  },
] as const

/** Spacing must come only from <Field>; a stacked utility would add to its gap. */
const STACKING_SPACING = /(^|\s)(space-y-|mt-|mb-|gap-)/

describe('field standard', () => {
  describe.each(FIELDS)('$name', ({ render: renderField }) => {
    it('wraps in the shared Field primitive', () => {
      const { container } = render(renderField({}))
      expect(container.querySelector('[data-slot="field"]')).toBeInTheDocument()
    })

    it('adds no spacing utility that would stack on Field\'s gap', () => {
      const { container } = render(renderField({}))
      const field = container.querySelector('[data-slot="field"]')
      const own = (field?.getAttribute('class') ?? '')
        .split(/\s+/)
        .filter((c) => STACKING_SPACING.test(` ${c}`))
        // Field's own `gap-3` is the standard; anything else is drift.
        .filter((c) => c !== 'gap-3')
      expect(own).toEqual([])
    })

    it('renders errors through FieldError, never a literal red', () => {
      const { container } = render(renderField({ error: 'Something is wrong' }))
      const error = container.querySelector('[data-slot="field-error"]')

      expect(error).toBeInTheDocument()
      expect(error).toHaveAttribute('role', 'alert')
      expect(error).toHaveTextContent('Something is wrong')
      // text-red-500/600 don't adapt to a branded or dark theme.
      expect(error?.className).not.toMatch(/text-red-\d/)
      expect(error?.className).toMatch(/text-destructive/)
    })

    it('marks the control invalid and points it at the error', () => {
      const { container } = render(renderField({ error: 'Bad' }))
      const invalid = container.querySelector('[aria-invalid="true"]')

      expect(invalid).toBeInTheDocument()
      const describedBy = invalid?.getAttribute('aria-describedby')
      expect(describedBy).toBeTruthy()
      expect(container.querySelector(`#${describedBy}`)).toHaveTextContent('Bad')
    })
  })

  // The consent checkbox has no text label of its own, so it only participates
  // in the error/aria half of the contract.
  describe('FormConsentCheckbox', () => {
    it('wraps in Field and reports errors like every other field', () => {
      const { container } = render(<FormConsentCheckbox error="You must accept" />)
      const error = container.querySelector('[data-slot="field-error"]')

      expect(container.querySelector('[data-slot="field"]')).toBeInTheDocument()
      expect(error).toHaveAttribute('role', 'alert')
      expect(error?.className).not.toMatch(/text-red-\d/)
      expect(container.querySelector('[aria-invalid="true"]')).toBeInTheDocument()
    })
  })

  describe('shared behaviour', () => {
    it('hides the description while an error is showing', () => {
      const { rerender } = render(
        <FormInputField label="Name" description="Your full name" />,
      )
      expect(screen.getByText('Your full name')).toBeInTheDocument()

      rerender(<FormInputField label="Name" description="Your full name" error="Required" />)
      expect(screen.queryByText('Your full name')).not.toBeInTheDocument()
      expect(screen.getByText('Required')).toBeInTheDocument()
    })

    it('uses one required marker across fields', () => {
      const { container: input } = render(<FormInputField label="Name" required />)
      const { container: password } = render(<FormPasswordField label="Password" required />)

      for (const c of [input, password]) {
        const marker = c.querySelector('label span')
        expect(marker).toHaveTextContent('*')
        expect(marker?.className).toMatch(/text-destructive/)
      }
    })

    it('puts labelAction on the label row, above the control', () => {
      const { container } = render(
        <FormPasswordField label="Password" labelAction={<a href="/forgot">Forgot?</a>} />,
      )
      const action = screen.getByText('Forgot?')
      const control = container.querySelector('input')

      expect(action.parentElement?.className).toContain('justify-between')
      expect(action.compareDocumentPosition(control!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    })
  })
})
