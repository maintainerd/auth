import { describe, expect, it } from 'vitest'
import { brandingSchema } from './brandingSchema'

const validBranding = {
  name: 'acme-light',
  layout: 'centered',
  ui_template: 'centered-card',
  company_name: 'Acme Inc.',
  logo_url: '',
  favicon_url: '',
  support_url: '',
  privacy_policy_url: '',
  terms_of_service_url: '',
}

describe('brandingSchema', () => {
  it.each(['centered', 'full_page', 'split'])('accepts the %s layout', async (layout) => {
    await expect(brandingSchema.validate({ ...validBranding, layout })).resolves.toMatchObject({ layout })
  })

  it('rejects an unsupported layout', async () => {
    await expect(
      brandingSchema.validate({ ...validBranding, layout: 'sidebar' }),
    ).rejects.toThrow('Select a valid login layout')
  })

  it('defaults an omitted layout to centered', async () => {
    const { layout } = await brandingSchema.validate({ ...validBranding, layout: undefined })
    expect(layout).toBe('centered')
  })

  it('accepts the configured login template', async () => {
    await expect(
      brandingSchema.validate({ ...validBranding, ui_template: 'split-showcase' }),
    ).resolves.toMatchObject({ ui_template: 'split-showcase' })
  })

  it('defaults an omitted login template to centered card', async () => {
    const { ui_template } = await brandingSchema.validate({ ...validBranding, ui_template: undefined })
    expect(ui_template).toBe('centered-card')
  })

  it('rejects an unsupported login template', async () => {
    await expect(
      brandingSchema.validate({ ...validBranding, ui_template: 'unknown-template' }),
    ).rejects.toThrow('Select a valid login template')
  })
})
