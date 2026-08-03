/**
 * Setup Hooks
 * Custom hooks for setup operations (tenant, admin, profile)
 */
import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useToast } from '@/hooks/useToast'
import {
  createTenantWithDefaults,
  createAdmin,
  createProfile,
  completeSetup,
  getSetupStatus,
} from '@/services'
import type { CreateAdminRequest, SetupStatusData } from '@/services/api/setup/types'

export function useSetupStatus() {
  const [status, setStatus] = useState<SetupStatusData | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  // Failures were swallowed into a null status, and the setup pages render
  // nothing while status is null — so a backend blip during first-run left the
  // user on a permanently blank white screen with no message and no retry.
  const [isError, setIsError] = useState(false)

  const checkStatus = useCallback(async () => {
    setIsLoading(true)
    setIsError(false)
    try {
      const response = await getSetupStatus()
      if (response.success && response.data) {
        setStatus(response.data)
        return response.data
      }
      setIsError(true)
      return null
    } catch {
      setIsError(true)
      return null
    } finally {
      setIsLoading(false)
    }
  }, [])

  return { status, isLoading, isError, checkStatus }
}

export function useSetupTenant() {
  const navigate = useNavigate()
  const { showError, showSuccess } = useToast()
  const [isLoading, setIsLoading] = useState(false)

  const createTenantWithDefaultsHook = useCallback(
    async (name: string, display_name: string, description?: string) => {
      setIsLoading(true)
      try {
        const response = await createTenantWithDefaults(name, display_name, description || '')
        showSuccess('Tenant created successfully!')
        navigate('/setup/admin')
        return { success: true, data: response }
      } catch (error: unknown) {
        if (error instanceof Error) {
          showError(error, 'Failed to create tenant')
          return { success: false, message: error.message }
        }
        showError('Unknown error', 'Failed to create tenant')
        return { success: false, message: 'Unknown error' }
      } finally {
        setIsLoading(false)
      }
    },
    [navigate, showError, showSuccess],
  )

  return { isLoading, createTenantWithDefaults: createTenantWithDefaultsHook }
}

export function useSetupAdmin() {
  const navigate = useNavigate()
  const { showError, showSuccess } = useToast()
  const [isLoading, setIsLoading] = useState(false)

  // Setup is a THREE-step server-side sequence, not two: create_admin,
  // create_profile, then complete. Only complete flips the tenant from
  // `pending` to `active`, and AuthEndpointTenantStatusMiddleware rejects every
  // login against a non-active tenant with 403 "tenant_unavailable".
  //
  // This used to stop after create_admin and navigate away, so running the
  // wizard to the end produced a tenant nobody could ever sign in to — the
  // button says "Complete setup" and it genuinely did not. /setup/complete
  // additionally refuses to run unless IsProfileSetup is true, which is why the
  // profile step is mandatory rather than cosmetic.
  const createAdminAccount = useCallback(
    async (data: { fullName: string; email: string; password: string }) => {
      setIsLoading(true)
      try {
        const fullName = data.fullName.trim()
        const adminData: CreateAdminRequest = {
          username: data.email,
          fullname: fullName,
          password: data.password,
          email: data.email,
        }
        await createAdmin(adminData)

        // Split on the first space: everything after it is the surname. The
        // backend only requires first_name, so a mononym is fine.
        const [firstName, ...rest] = fullName.split(/\s+/)
        const lastName = rest.join(' ')
        await createProfile({
          first_name: firstName,
          ...(lastName ? { last_name: lastName } : {}),
          email: data.email,
        })

        await completeSetup()

        showSuccess('Setup completed successfully!')
        navigate('/')
        return { success: true }
      } catch (error: unknown) {
        if (error instanceof Error) {
          showError(error, 'Failed to complete setup')
          return { success: false, message: error.message }
        }
        showError('Unknown error', 'Failed to complete setup')
        return { success: false, message: 'Unknown error' }
      } finally {
        setIsLoading(false)
      }
    },
    [navigate, showError, showSuccess],
  )

  return { isLoading, createAdminAccount }
}

export function useCompleteSetup() {
  const { showError, showSuccess } = useToast()
  const [isLoading, setIsLoading] = useState(false)

  const finalizeSetup = useCallback(async () => {
    setIsLoading(true)
    try {
      await completeSetup()
      showSuccess('Setup completed successfully!')
      return { success: true }
    } catch (error: unknown) {
      if (error instanceof Error) {
        showError(error, 'Failed to complete setup')
        return { success: false, message: error.message }
      }
      showError('Unknown error', 'Failed to complete setup')
      return { success: false, message: 'Unknown error' }
    } finally {
      setIsLoading(false)
    }
  }, [showError, showSuccess])

  return { isLoading, finalizeSetup }
}
