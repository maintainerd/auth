import { describe, it, expect } from 'vitest'
import { buildLoginSchema } from './authSchema'

/**
 * Sign-in accepts a USERNAME or an email. The backend looks the identifier up by
 * username first and falls back to email (authn.service_login), and usernames
 * are only length-checked — so admin-created accounts routinely have identifiers
 * that are not email-shaped.
 *
 * This schema enforced .email(), which rejected those before the request was
 * ever sent. The field was labelled "Email", so there was no clue either.
 */
describe('login identifier validation', () => {
  const schema = buildLoginSchema()
  const check = (email: string) => schema.validateAt('email', { email, password: 'x' })

  it('accepts a plain username', async () => {
    await expect(check('ada')).resolves.toBeDefined()
    await expect(check('ada.lovelace')).resolves.toBeDefined()
    await expect(check('admin_1')).resolves.toBeDefined()
  })

  it('still accepts an email address', async () => {
    await expect(check('ada@example.com')).resolves.toBeDefined()
  })

  it('rejects an empty identifier', async () => {
    await expect(check('')).rejects.toThrow(/required/i)
  })

  it('rejects an over-length identifier', async () => {
    await expect(check('a'.repeat(256))).rejects.toThrow(/exceed/i)
  })
})
