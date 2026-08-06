import { beforeEach, describe, expect, it, vi } from 'vitest'
import { get, put } from '../client'
import { changePassword, fetchAccountInfo, type AccountInfo } from './index'

vi.mock('../client', async () => {
  const actual = await vi.importActual<typeof import('../client')>('../client')
  return {
    ...actual,
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    deleteRequest: vi.fn(),
  }
})

// The exact body internal/user/handler_account.go emits for GET /account.
const accountResponse = {
  user_id: '11111111-1111-1111-1111-111111111111',
  email: 'ada@example.com',
  phone: '',
  email_verified: true,
  phone_verified: false,
  profiles: [{ profile_id: 'profile-1', first_name: 'Ada', display_name: 'Ada L', default: true }],
  roles: ['member'],
  permissions: ['account:profile:read:self'],
  tenant: { tenant_id: 'tenant-1', name: 'acme', display_name: 'Acme', identifier: 'acme' },
}

describe('fetchAccountInfo', () => {
  beforeEach(() => {
    vi.mocked(get).mockReset()
  })

  it('returns the fields the endpoint actually sends', async () => {
    vi.mocked(get).mockResolvedValue({ success: true, data: accountResponse })

    const account: AccountInfo = await fetchAccountInfo()

    expect(account.user_id).toBe('11111111-1111-1111-1111-111111111111')
    expect(account.email_verified).toBe(true)
    expect(account.profiles).toHaveLength(1)
  })

  // A 401, a 429 and a 500 used to be flattened into "no account data", which
  // left dependent controls disabled with nothing on screen to explain it.
  it('propagates transport failures instead of reporting an empty account', async () => {
    vi.mocked(get).mockRejectedValue(new Error('Too many requests'))

    await expect(fetchAccountInfo()).rejects.toThrow('Too many requests')
  })

  it('rejects an envelope with no account in it', async () => {
    vi.mocked(get).mockResolvedValue({ success: false, message: 'Unauthorized' })

    await expect(fetchAccountInfo()).rejects.toThrow('Unauthorized')
  })
})

describe('changePassword', () => {
  beforeEach(() => {
    vi.mocked(put).mockReset()
  })

  it('reports the server verdict verbatim', async () => {
    vi.mocked(put).mockResolvedValue({
      success: true,
      data: { other_sessions_revoked: true, reauthentication_required: true },
    })

    await expect(changePassword('old', 'new')).resolves.toEqual({
      other_sessions_revoked: true,
      reauthentication_required: true,
    })
  })

  // client.ts turns an empty 200 into `{ success: true }`. Defaulting the missing
  // body to all-false let the UI say "Password changed successfully" for a
  // response that never said so — and the user was then silently signed out.
  it('refuses to invent a verdict when the response carries no body', async () => {
    vi.mocked(put).mockResolvedValue({ success: true })

    await expect(changePassword('old', 'new')).rejects.toThrow(/could not confirm it/)
  })
})
