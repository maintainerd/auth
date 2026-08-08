import { describe, expect, it } from 'vitest'
import { buildProfilePayload, validateNotCleared } from './profilePayload'

const values = {
  display_name: 'Ada L',
  first_name: 'Ada',
  middle_name: 'Augusta',
  last_name: 'Lovelace',
  gender: 'female',
  birthdate: '1815-12-10',
  profile_url: 'https://cdn.example.com/ada.png',
}

describe('buildProfilePayload', () => {
  it('sends every filled-in field on a create', () => {
    expect(buildProfilePayload(values)).toEqual({
      first_name: 'Ada',
      display_name: 'Ada L',
      middle_name: 'Augusta',
      last_name: 'Lovelace',
      gender: 'female',
      birthdate: '1815-12-10',
      profile_url: 'https://cdn.example.com/ada.png',
    })
  })

  it('omits blank optional fields instead of sending "" (NilOrNotEmpty rejects "")', () => {
    const payload = buildProfilePayload({
      display_name: '',
      first_name: 'Ada',
      middle_name: '',
      last_name: '   ',
      gender: '',
      birthdate: '',
      profile_url: '',
    })

    expect(payload).toEqual({ first_name: 'Ada' })
    expect(Object.values(payload)).not.toContain('')
  })

  it('trims the values it does send', () => {
    const payload = buildProfilePayload({
      display_name: '  Ada L  ',
      first_name: '  Ada  ',
      middle_name: '  Augusta  ',
      last_name: '  Lovelace  ',
      gender: '  female  ',
      birthdate: '  1815-12-10  ',
      profile_url: '  https://cdn.example.com/ada.png  ',
    })

    expect(payload.first_name).toBe('Ada')
    expect(payload.display_name).toBe('Ada L')
    expect(payload.middle_name).toBe('Augusta')
    expect(payload.last_name).toBe('Lovelace')
    expect(payload.gender).toBe('female')
    expect(payload.birthdate).toBe('1815-12-10')
    expect(payload.profile_url).toBe('https://cdn.example.com/ada.png')
  })

  it('sends only the changed field on an edit, so an untouched one cannot be clobbered', () => {
    const payload = buildProfilePayload(values, { display_name: true })

    expect(payload).toEqual({ first_name: 'Ada', display_name: 'Ada L' })
    expect('middle_name' in payload).toBe(false)
    expect('last_name' in payload).toBe(false)
    expect('gender' in payload).toBe(false)
    expect('birthdate' in payload).toBe(false)
    expect('profile_url' in payload).toBe(false)
  })

  it('sends middle_name, gender and birthdate once the user changed them', () => {
    const payload = buildProfilePayload(values, { middle_name: true, gender: true, birthdate: true })

    expect(payload).toEqual({ first_name: 'Ada', middle_name: 'Augusta', gender: 'female', birthdate: '1815-12-10' })
  })

  it('always sends first_name, which the DTO marks Required', () => {
    expect(buildProfilePayload(values, {})).toEqual({ first_name: 'Ada' })
  })

  it('never sends fields outside this form, so the server keeps them', () => {
    const payload = buildProfilePayload(values, { display_name: true, last_name: true })

    for (const field of ['email', 'timezone', 'language', 'metadata']) {
      expect(field in payload).toBe(false)
    }
  })
})

describe('validateNotCleared', () => {
  it('rejects blanking a field that has a stored value, because the DTO cannot express a clear', () => {
    expect(validateNotCleared('Display name', 'Ada L')('')).toBe(
      "Display name can't be removed here yet. Enter a value, or restore the previous one.",
    )
    expect(validateNotCleared('Display name', 'Ada L')('   ')).toBeTypeOf('string')
  })

  it('allows a blank field that was already blank', () => {
    expect(validateNotCleared('Display name', undefined)('')).toBe(true)
    expect(validateNotCleared('Display name', '')('')).toBe(true)
  })

  it('allows any non-empty value', () => {
    expect(validateNotCleared('Display name', 'Ada L')('Ada Lovelace')).toBe(true)
  })
})
