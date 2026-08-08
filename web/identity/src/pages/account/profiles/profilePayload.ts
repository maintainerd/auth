import type { ProfileRequest } from '@/services/api/account'

/** The subset of a profile this form actually lets the user edit. */
export interface ProfileFormValues {
  display_name: string
  first_name: string
  middle_name: string
  last_name: string
  gender: string
  birthdate: string
  profile_url: string
}

/** react-hook-form's `dirtyFields` for this form — true per field the user changed. */
export type ProfileFormDirtyFields = Partial<Record<keyof ProfileFormValues, boolean>>

/**
 * Trimmed value, or undefined when there is nothing to send.
 *
 * The distinction matters on the wire. Every optional field on
 * user.ProfileRequestDTO is a *string, and its rule is NilOrNotEmpty: absent
 * (nil) passes, `""` decodes to a pointer-to-empty-string and FAILS. Sending
 * `"display_name": ""` for a field the user simply left blank produced a 400
 * listing "cannot be blank" for fields the form never marked required.
 */
function optional(value?: string | null): string | undefined {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}

/**
 * Builds the profile body for a create or an edit.
 *
 * On an edit only the fields the user actually changed are sent.
 * applyProfileFields (internal/user/service_profile.go) skips every nil pointer,
 * so an omitted key means "leave this alone" and an untouched field — including
 * the OIDC address claim in metadata, which no form renders — survives.
 * Echoing the loaded profile back instead would also work, but it silently
 * reverts anything changed on another device between the read and the write;
 * not sending a field cannot lose that race.
 *
 * first_name is always sent: the DTO marks it Required, so it is not omissible.
 *
 * Pass no dirtyFields for a create, where there is nothing stored to preserve
 * and every filled-in field is new.
 */
export function buildProfilePayload(
  values: ProfileFormValues,
  dirtyFields?: ProfileFormDirtyFields,
): ProfileRequest {
  const changed = (field: keyof ProfileFormValues) => !dirtyFields || dirtyFields[field] === true

  const payload: ProfileRequest = { first_name: values.first_name.trim() }
  if (changed('display_name')) payload.display_name = optional(values.display_name)
  if (changed('middle_name')) payload.middle_name = optional(values.middle_name)
  if (changed('last_name')) payload.last_name = optional(values.last_name)
  if (changed('gender')) payload.gender = optional(values.gender)
  if (changed('birthdate')) payload.birthdate = optional(values.birthdate)
  if (changed('profile_url')) payload.profile_url = optional(values.profile_url)
  return payload
}

/**
 * Rejects blanking out a field that already has a stored value.
 *
 * There is no way to express "clear this" in ProfileRequestDTO: `""` is
 * rejected by NilOrNotEmpty and an omitted key is read as "unchanged". A blanked
 * field would therefore save, report success, and come back with the old value
 * still on it. Saying so in the field is the honest alternative to a success
 * toast that did nothing.
 */
export function validateNotCleared(label: string, storedValue?: string | null) {
  return (value: string): string | true => {
    if (value.trim() || !storedValue?.trim()) return true
    return `${label} can't be removed here yet. Enter a value, or restore the previous one.`
  }
}
