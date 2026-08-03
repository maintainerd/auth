import { beforeEach, describe, expect, it, vi } from 'vitest'
import { bootstrapToTenantEntity, fetchTenantBootstrap } from './index'
import { get } from '../client'
import type { TenantBootstrap } from './types'

// Mock the HTTP client so importing the service module doesn't pull in the
// dev-only debug helper (which touches `window` after teardown).
vi.mock('../client', () => ({ get: vi.fn() }))
const getMock = vi.mocked(get)

const bootstrap: TenantBootstrap = {
  tenant: {
    tenant_uuid: 'uuid-1',
    name: 'acme',
    display_name: 'Acme',
    description: 'Acme tenant',
    status: 'active',
    is_system: false,
  },
  surface: 'identity',
  identity_url: 'https://acme.auth.maintainerd.local',
  console_url: 'https://acme.console.auth.maintainerd.local',
  password_config: {
    min_length: 10,
    max_length: 64,
    require_uppercase: true,
    require_lowercase: true,
    require_number: true,
    require_symbol: false,
    min_strength_score: 0,
    reject_common_passwords: true,
    check_hibp: false,
  },
  registration_config: {
    self_registration_enabled: false,
    require_email_verification: true,
    captcha_on_signup: true,
  },
  branding: {
    layout: 'centered',
    company_name: 'Acme',
    logo_url: '',
    favicon_url: '',
    support_url: '',
    privacy_policy_url: '',
    terms_of_service_url: '',
    metadata: null,
  },
  client: { client_id: 'acme-identity', name: 'identity', display_name: 'Identity', client_type: 'public' },
}

describe('bootstrapToTenantEntity', () => {
  it('carries the tenant identity, slug and branding through', () => {
    const entity = bootstrapToTenantEntity(bootstrap)
    expect(entity.tenant_id).toBe('uuid-1')
    expect(entity.name).toBe('acme')
    expect(entity.display_name).toBe('Acme')
    expect(entity.is_system).toBe(false)
    expect(entity.branding?.company_name).toBe('Acme')
  })

  it('carries the tenant password_config through in the public shape', () => {
    const entity = bootstrapToTenantEntity(bootstrap)
    expect(entity.password_config).toEqual({
      min_length: 10,
      max_length: 64,
      require_uppercase: true,
      require_lowercase: true,
      require_number: true,
      require_symbol: false,
      min_strength_score: 0,
      reject_common_passwords: true,
      check_hibp: false,
    })
  })

  it('carries the tenant registration_config through in the public shape', () => {
    const entity = bootstrapToTenantEntity(bootstrap)
    expect(entity.registration_config).toEqual({
      self_registration_enabled: false,
      require_email_verification: true,
      captcha_on_signup: true,
    })
  })

  it('leaves config undefined when the bootstrap omits it', () => {
    const entity = bootstrapToTenantEntity({ ...bootstrap, password_config: undefined, registration_config: undefined })
    expect(entity.password_config).toBeUndefined()
    expect(entity.registration_config).toBeUndefined()
  })
})

describe('fetchTenantBootstrap', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('sends host and client_id to the domain bootstrap endpoint', async () => {
    getMock.mockResolvedValueOnce({ success: true, data: bootstrap, message: '' })

    await expect(fetchTenantBootstrap('acme.auth.maintainerd.local', ' client-abc ')).resolves.toEqual(bootstrap)

    expect(getMock).toHaveBeenCalledWith('/tenant?domain=acme.auth.maintainerd.local&client_id=client-abc')
  })

  it('omits client_id when it is not provided', async () => {
    getMock.mockResolvedValueOnce({ success: true, data: bootstrap, message: '' })

    await expect(fetchTenantBootstrap('acme.auth.maintainerd.local')).resolves.toEqual(bootstrap)

    expect(getMock).toHaveBeenCalledWith('/tenant?domain=acme.auth.maintainerd.local')
  })
})
