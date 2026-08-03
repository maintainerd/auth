import { render, screen } from '@testing-library/react'
import { Plus } from 'lucide-react'
import { describe, expect, it } from 'vitest'
import { Button } from '@/components/ui/button'
import { SettingsCard } from './SettingsCard'

describe('SettingsCard', () => {
  it('renders a reusable header action beside the title block', () => {
    render(
      <SettingsCard
        title="Profiles"
        description="Manage account profiles."
        icon={Plus}
        action={<Button type="button">Add profile</Button>}
      >
        <p>Profile rows</p>
      </SettingsCard>,
    )

    expect(screen.getByText('Profiles')).toBeInTheDocument()
    expect(screen.getByText('Manage account profiles.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add profile' })).toBeInTheDocument()
    expect(screen.getByText('Profile rows')).toBeInTheDocument()
  })
})
