import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import ProfileFormPage from './ProfileFormPage'

const { fetchProfilesMock, updateProfileMock, createProfileMock, navigateMock } = vi.hoisted(() => ({
  fetchProfilesMock: vi.fn(),
  updateProfileMock: vi.fn(),
  createProfileMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock('@/services/api/account', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/account')>('@/services/api/account')
  return {
    ...actual,
    fetchProfiles: fetchProfilesMock,
    updateProfile: updateProfileMock,
    createProfile: createProfileMock,
  }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ logout: vi.fn().mockResolvedValue(undefined) }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({ currentTenant: null }),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => navigateMock }
})

const storedProfile = {
  profile_id: 'profile-1',
  display_name: 'Ada L',
  first_name: 'Ada',
  middle_name: 'Augusta',
  last_name: 'Lovelace',
  birthdate: '1815-12-10T00:00:00Z',
  gender: 'female',
  email: 'ada@example.com',
  timezone: 'Europe/London',
  language: 'en-US',
  metadata: { address: { locality: 'London' } },
  is_default: true,
  created_at: '2024-01-01T00:00:00Z',
}

function renderEdit() {
  return renderWithProviders(<ProfileFormPage />, {
    route: '/account/profile/profile-1/edit',
    path: '/account/profile/:profileId/edit',
  })
}

describe('ProfileFormPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchProfilesMock.mockResolvedValue([storedProfile])
    updateProfileMock.mockResolvedValue(storedProfile)
    createProfileMock.mockResolvedValue(storedProfile)
  })

  // The server merges partial updates (applyProfileFields skips nil pointers),
  // so the edited field is all that needs to travel. Resending the loaded
  // profile would revert whatever another device changed in the meantime.
  it('sends only the edited field, leaving the rest for the server to keep', async () => {
    const user = userEvent.setup()
    renderEdit()

    const displayName = await screen.findByLabelText('Display name')
    await waitFor(() => expect(displayName).toHaveValue('Ada L'))

    await user.clear(displayName)
    await user.type(displayName, 'Ada Lovelace')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(updateProfileMock).toHaveBeenCalledTimes(1))
    expect(updateProfileMock).toHaveBeenCalledWith('profile-1', {
      first_name: 'Ada',
      display_name: 'Ada Lovelace',
    })
  })

  // A blanked field is a silent no-op server-side: "" fails NilOrNotEmpty and an
  // omitted key means "unchanged", so the save would report success and change
  // nothing.
  it('refuses to blank a stored value instead of reporting a save that did nothing', async () => {
    const user = userEvent.setup()
    renderEdit()

    const displayName = await screen.findByLabelText('Display name')
    await waitFor(() => expect(displayName).toHaveValue('Ada L'))

    await user.clear(displayName)
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(
      await screen.findByText("Display name can't be removed here yet. Enter a value, or restore the previous one."),
    ).toBeInTheDocument()
    expect(updateProfileMock).not.toHaveBeenCalled()
  })

  it('enforces the server-required first name before sending anything', async () => {
    const user = userEvent.setup()
    fetchProfilesMock.mockResolvedValue([])
    renderWithProviders(<ProfileFormPage />, {
      route: '/account/profile/new',
      path: '/account/profile/new',
    })

    await user.type(screen.getByLabelText('Last name'), 'Lovelace')
    await user.click(screen.getByRole('button', { name: 'Create profile' }))

    expect(await screen.findByText('First name is required.')).toBeInTheDocument()
    expect(createProfileMock).not.toHaveBeenCalled()
  })

  // A scheme-less value is already stopped by the field's native type="url"
  // constraint; this covers the part only the resolver can catch — a URL the
  // browser accepts but an <img src> could never load.
  it('rejects an avatar URL that is not http(s) before the round trip', async () => {
    const user = userEvent.setup()
    fetchProfilesMock.mockResolvedValue([])
    renderWithProviders(<ProfileFormPage />, {
      route: '/account/profile/new',
      path: '/account/profile/new',
    })

    await user.type(screen.getByLabelText(/First name/), 'Ada')
    await user.type(screen.getByLabelText('Avatar URL'), 'ftp://cdn.example.com/ada.png')
    await user.click(screen.getByRole('button', { name: 'Create profile' }))

    expect(await screen.findByText('Enter a full URL, starting with https://')).toBeInTheDocument()
    expect(createProfileMock).not.toHaveBeenCalled()
  })

  it('omits fields the user left blank rather than sending empty strings', async () => {
    const user = userEvent.setup()
    fetchProfilesMock.mockResolvedValue([])
    renderWithProviders(<ProfileFormPage />, {
      route: '/account/profile/new',
      path: '/account/profile/new',
    })

    await user.type(screen.getByLabelText(/First name/), 'Ada')
    await user.click(screen.getByRole('button', { name: 'Create profile' }))

    await waitFor(() => expect(createProfileMock).toHaveBeenCalledTimes(1))
    const payload = createProfileMock.mock.calls[0][0]
    expect(payload.first_name).toBe('Ada')
    expect(payload.display_name).toBeUndefined()
    expect(payload.last_name).toBeUndefined()
    expect(payload.profile_url).toBeUndefined()
  })
})
