import { useNavigate } from 'react-router-dom'
import { AtSign, KeyRound, Mail } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { SettingsActionRow } from '@/components/settings'

/**
 * Security hub.
 *
 * Each credential change is its own route rather than a modal. A dialog cannot
 * be linked to, survives neither a reload nor the browser back button, and
 * cramps the password form's live policy checklist — on a phone a modal taller
 * than the viewport is worse still. The forms share SecurityFormPage so their
 * chrome and responsive behaviour stay identical.
 */
export default function AccountSecurityPage() {
  const navigate = useNavigate()

  return (
    <AccountLayout title="Security">
      <SettingsCard
        title="Sign-in details"
        description="Manage the identifiers and password used to access your account."
        icon={AtSign}
        contentClassName="space-y-2"
      >
        <div className="space-y-2">
          <SettingsActionRow
            icon={Mail}
            title="Email address"
            description="Change the email address you use for sign-in."
            actionLabel="Change"
            onAction={() => navigate('/account/security/email')}
          />
          <SettingsActionRow
            icon={AtSign}
            title="Username"
            description="Update the username attached to your account."
            actionLabel="Change"
            onAction={() => navigate('/account/security/username')}
          />
          <SettingsActionRow
            icon={KeyRound}
            title="Password"
            description="Change your password without leaving your account."
            actionLabel="Change"
            onAction={() => navigate('/account/security/password')}
          />
        </div>
      </SettingsCard>
    </AccountLayout>
  )
}
