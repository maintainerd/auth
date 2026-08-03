import { lazy, Suspense } from 'react'
import { Route, Routes, Navigate } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { StepUpProvider } from './components/stepup/StepUpProvider'
import { ToastContainer } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'
import '@/styles/toast.css'
import { queryClient } from '@/lib/queryClient'
import { AppBootstrap } from './components/auth/AppBootstrap'
import ErrorBoundary from './components/ErrorBoundary'
import AppLoadingScreen from './components/layout/AppLoadingScreen'

// Route pages are code-split with React.lazy so the initial bundle only carries
// the app shell; each route's chunk is fetched on demand. <Suspense> shows the
// existing bootstrap splash while a chunk loads.
const NoAccessPage = lazy(() => import('./pages/no-access/NoAccessPage'))
const ServiceUnavailablePage = lazy(() => import('./pages/service-unavailable/ServiceUnavailablePage'))
const LoginPage = lazy(() => import('./pages/login'))
const RegisterPage = lazy(() => import('./pages/register'))
const RegisterInvitePage = lazy(() => import('./pages/register/invite/RegisterInvitePage'))
const ForgotPasswordPage = lazy(() => import('./pages/forgot-password'))
const ResetPasswordPage = lazy(() => import('./pages/reset-password'))
const MagicLinkPage = lazy(() => import('./pages/magic-link/MagicLinkPage'))
const RegisterProfilePage = lazy(() => import('./pages/register/profile'))
const VerifyEmailPage = lazy(() => import('./pages/register/verify-email/VerifyEmailPage'))
const LoginSuccessPage = lazy(() => import('./pages/login-success'))
const OAuthAuthorizePage = lazy(() => import('./pages/oauth/authorize/OAuthAuthorizePage'))
const OAuthConsentPage = lazy(() => import('./pages/oauth/consent/OAuthConsentPage'))
const OAuthDevicePage = lazy(() => import('./pages/oauth/device/OAuthDevicePage'))
const OAuthCIBAPage = lazy(() => import('./pages/oauth/ciba/OAuthCIBAPage'))
const OAuthGrantsPage = lazy(() => import('./pages/oauth/grants/OAuthGrantsPage'))
const OAuthEndSessionPage = lazy(() => import('./pages/oauth/end-session/OAuthEndSessionPage'))
const MFAPage = lazy(() => import('./pages/account/mfa'))
const MFATotpPage = lazy(() => import('./pages/account/mfa/TOTPSetupPage'))
const MFAPasskeyPage = lazy(() => import('./pages/account/mfa/PasskeySetupPage'))
const MFASmsPage = lazy(() => import('./pages/account/mfa/SMSSetupPage'))
const MFAEmailOtpPage = lazy(() => import('./pages/account/mfa/EmailOtpSetupPage'))
const SMSLoginPage = lazy(() => import('./pages/sms-login/SMSLoginPage'))
const LinkedIdentitiesPage = lazy(() => import('./pages/account/identities/LinkedIdentitiesPage'))
const VerifyPhonePage = lazy(() => import('./pages/account/phone/VerifyPhonePage'))
const BackupCodeRecoveryPage = lazy(() => import('./pages/recovery/BackupCodeRecoveryPage'))
const AccountLockedPage = lazy(() => import('./pages/account-locked/AccountLockedPage'))
const TooManyRequestsPage = lazy(() => import('./pages/too-many-requests/TooManyRequestsPage'))
const NotFoundPage = lazy(() => import('./pages/not-found/NotFoundPage'))
const AccountLinkPage = lazy(() => import('./pages/account-link/AccountLinkPage'))
const ErasurePage = lazy(() => import('./pages/account/erasure/ErasurePage'))
const AccountProfilesPage = lazy(() => import('./pages/account/profiles/AccountProfilesPage'))
const AccountProfileFormPage = lazy(() => import('./pages/account/profiles/ProfileFormPage'))
const AccountSecurityPage = lazy(() => import('./pages/account/security/AccountSecurityPage'))
const AccountSessionsPage = lazy(() => import('./pages/account/sessions/AccountSessionsPage'))
const AccountDevicesPage = lazy(() => import('./pages/account/devices/AccountDevicesPage'))
const AccountSettingsPage = lazy(() => import('./pages/account/settings/AccountSettingsPage'))
const AccountDataPage = lazy(() => import('./pages/account/data/AccountDataPage'))

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <StepUpProvider>
      <AppBootstrap>
      <ErrorBoundary>
      <Suspense fallback={<AppLoadingScreen />}>
      <Routes>
        <Route path="/" element={<Navigate to="/login" replace />} />
        <Route path="/no-access" element={<NoAccessPage />} />
        <Route path="/service-unavailable" element={<ServiceUnavailablePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/register/invite" element={<RegisterInvitePage />} />
        <Route path="/email-verification" element={<VerifyEmailPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/magic-link" element={<MagicLinkPage />} />
        <Route path="/authorize" element={<OAuthAuthorizePage />} />
        <Route path="/oauth/authorize" element={<OAuthAuthorizePage />} />
        <Route path="/oauth/consent/:challengeId" element={<OAuthConsentPage />} />
        <Route path="/device" element={<OAuthDevicePage />} />
        <Route path="/oauth/device" element={<OAuthDevicePage />} />
        <Route path="/ciba" element={<OAuthCIBAPage />} />
        <Route path="/oauth/ciba" element={<OAuthCIBAPage />} />
        <Route path="/oauth/grants" element={<OAuthGrantsPage />} />
        <Route path="/oauth/end_session" element={<OAuthEndSessionPage />} />
        <Route path="/oauth/end-session" element={<OAuthEndSessionPage />} />
        <Route path="/register/profile" element={<RegisterProfilePage />} />
        <Route path="/login-success" element={<LoginSuccessPage />} />
        <Route path="/account/mfa" element={<MFAPage />} />
        <Route path="/account/mfa/totp" element={<MFATotpPage />} />
        <Route path="/account/mfa/passkeys" element={<MFAPasskeyPage />} />
        <Route path="/account/mfa/sms" element={<MFASmsPage />} />
        <Route path="/account/mfa/email-otp" element={<MFAEmailOtpPage />} />
        <Route path="/sms-login" element={<SMSLoginPage />} />
        <Route path="/account/identities" element={<LinkedIdentitiesPage />} />
        <Route path="/account/phone" element={<VerifyPhonePage />} />
        <Route path="/recovery" element={<BackupCodeRecoveryPage />} />
        <Route path="/account-locked" element={<AccountLockedPage />} />
        <Route path="/too-many-requests" element={<TooManyRequestsPage />} />
        <Route path="/account-link" element={<AccountLinkPage />} />
        <Route path="/account/erasure" element={<ErasurePage />} />
        <Route path="/account" element={<Navigate to="/account/profile" replace />} />
        <Route path="/account/profile" element={<AccountProfilesPage />} />
        <Route path="/account/profile/new" element={<AccountProfileFormPage />} />
        <Route path="/account/profile/:profileId/edit" element={<AccountProfileFormPage />} />
        <Route path="/account/security" element={<AccountSecurityPage />} />
        <Route path="/account/sessions" element={<AccountSessionsPage />} />
        <Route path="/account/devices" element={<AccountDevicesPage />} />
        <Route path="/account/settings" element={<AccountSettingsPage />} />
        <Route path="/account/data" element={<AccountDataPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      </Suspense>
      </ErrorBoundary>
      </AppBootstrap>
      <ToastContainer
        position="bottom-right"
        autoClose={5000}
        hideProgressBar={true}
        newestOnTop={false}
        closeOnClick
        rtl={false}
        pauseOnFocusLoss
        draggable
        pauseOnHover
        theme="light"
      />
      </StepUpProvider>
    </QueryClientProvider>
  )
}

export default App
