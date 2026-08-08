import { describe, expect, it } from 'vitest'
import { buildRegisterSchema } from './authSchema'

const base = {
  email: 'buyer@example.com',
  password: 'Str0ng!Passw0rd',
  confirmPassword: 'Str0ng!Passw0rd',
  acceptTerms: true,
}

describe('buildRegisterSchema conditional fields', () => {
  it('ignores full name and phone when the flow does not require them', async () => {
    const schema = buildRegisterSchema(undefined, {})
    await expect(schema.validate(base)).resolves.toBeTruthy()
  })

  it('requires full name only when the flow demands it', async () => {
    const optional = buildRegisterSchema(undefined, {})
    await expect(optional.validate(base)).resolves.toBeTruthy()

    const required = buildRegisterSchema(undefined, { fullname: true })
    await expect(required.validate(base)).rejects.toThrow(/full name is required/i)
    await expect(required.validate({ ...base, fullname: 'Ada Lovelace' })).resolves.toBeTruthy()
  })

  it('rejects a whitespace-only full name', async () => {
    const schema = buildRegisterSchema(undefined, { fullname: true })
    await expect(schema.validate({ ...base, fullname: '   ' })).rejects.toThrow(/full name is required/i)
  })

  it('caps full name at the backend limit', async () => {
    const schema = buildRegisterSchema(undefined, { fullname: true })
    await expect(
      schema.validate({ ...base, fullname: 'a'.repeat(256) }),
    ).rejects.toThrow(/not exceed 255/i)
  })

  // Parity with internal/platform/valid/valid.go IsValidPhoneNumber: the shape
  // regex plus a 7-15 digit count. Accepting anything it rejects would produce a
  // 400 the user cannot act on.
  describe('phone matches the backend rule exactly', () => {
    const schema = buildRegisterSchema(undefined, { phone: true })

    it.each([
      ['+1 212 555 1234', 'international with spaces'],
      ['+12125551234', 'international compact'],
      ['212-555-1234', 'hyphenated'],
      ['212.555.1234', 'dotted'],
    ])('accepts %s (%s)', async (phone) => {
      await expect(schema.validate({ ...base, phone })).resolves.toBeTruthy()
    })

    it.each([
      ['0123456789', 'leading zero is rejected by the backend regex'],
      ['123456', 'fewer than 7 digits'],
      ['1234567890123456', 'more than 15 digits'],
      ['not-a-phone', 'letters'],
      ['+', 'punctuation only'],
      // The backend regex requires [1-9] as the first character after an optional
      // '+', so a leading parenthesis is rejected despite valid.go's comment
      // claiming "(123) 456-7890" is accepted.
      ['(212) 555-1234', 'leading parenthesis'],
    ])('rejects %s (%s)', async (phone) => {
      await expect(schema.validate({ ...base, phone })).rejects.toThrow(/country code/i)
    })

    it('is required when demanded and blank', async () => {
      await expect(schema.validate(base)).rejects.toThrow(/phone number is required/i)
    })

    it('still validates format when optional but present', async () => {
      const optional = buildRegisterSchema(undefined, {})
      await expect(optional.validate({ ...base, phone: '0123' })).rejects.toThrow(/country code/i)
      await expect(optional.validate({ ...base, phone: '' })).resolves.toBeTruthy()
    })
  })
})
