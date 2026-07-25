import { lazy, Suspense } from 'react'
import { Route, Routes, Navigate } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { StepUpProvider } from './components/stepup/StepUpProvider'
import { ToastContainer } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'
import '@/styles/toast.css'
import { queryClient } from '@/lib/queryClient'
import { AppBootstrap } from './components/auth/AppBootstrap'
import { PrivateLayout } from './components/layout/PrivateLayout'
import AppLoadingScreen from './components/layout/AppLoadingScreen'
import ErrorBoundary from './components/ErrorBoundary'

// The app shell (bootstrap, layout, login landing) stays eager so the first
// paint is immediate. Every route page below is code-split via React.lazy so
// the initial bundle only carries the shell + the current route's chunk.
import LoginPage from './pages/login/LoginPage'
import OAuthCallbackPage from './pages/auth/OAuthCallbackPage'

const NoAccessPage = lazy(() => import('./pages/no-access/NoAccessPage'))
const ServiceUnavailablePage = lazy(() => import('./pages/service-unavailable/ServiceUnavailablePage'))
const SetupTenantPage = lazy(() => import('./pages/setup/tenant'))
const SetupAdminPage = lazy(() => import('./pages/setup/admin'))
const DashboardPage = lazy(() => import('./pages/dashboard'))
const UsersPage = lazy(() => import('./pages/users'))
const RolesPage = lazy(() => import('./pages/roles'))
const InvitationsPage = lazy(() => import('./pages/invitations'))
const UserDetailsPage = lazy(() => import('./pages/users/details'))
const InviteForm = lazy(() => import('./pages/invitations/form/InviteForm'))
const InviteDetailsPage = lazy(() => import('./pages/invitations/details/InviteDetailsPage'))
const UserAddOrUpdateForm = lazy(() => import('./pages/users/form'))
const ServicesPage = lazy(() => import('./pages/services'))
const ServiceDetailsPage = lazy(() => import('./pages/services/details'))
const ServiceAddOrUpdateForm = lazy(() => import('./pages/services/form'))
const ApisPage = lazy(() => import('./pages/apis'))
const ApiDetailsPage = lazy(() => import('./pages/apis/details'))
const ApiAddOrUpdateForm = lazy(() => import('./pages/apis/form'))
const PermissionsPage = lazy(() => import('./pages/permissions/PermissionsPage'))
const RoleDetailsPage = lazy(() => import('./pages/roles/details'))
const RoleAddOrUpdateForm = lazy(() => import('./pages/roles/form'))
const IdentityProvidersPage = lazy(() => import('./pages/identity-providers'))
const IdentityProviderDetailsPage = lazy(() => import('./pages/identity-providers/details'))
const IdentityProviderAddOrUpdateForm = lazy(() => import('./pages/identity-providers/form'))
const ClientsPage = lazy(() => import('./pages/clients'))
const ClientDetailsPage = lazy(() => import('./pages/clients/details'))
const ClientAddOrUpdateForm = lazy(() => import('./pages/clients/form'))
const EventsAndWebhooksPage = lazy(() => import('./pages/events/EventsAndWebhooksPage'))
const WebhookDetailsPage = lazy(() => import('./pages/webhooks/details'))
const WebhookAddOrUpdateForm = lazy(() => import('./pages/webhooks/form'))
const WorkloadIdentityPage = lazy(() => import('./pages/workload-identity'))
const WorkloadIdentityDetailsPage = lazy(() => import('./pages/workload-identity/details'))
const WorkloadIdentityAddOrUpdateForm = lazy(() => import('./pages/workload-identity/form'))
const PoliciesPage = lazy(() => import('./pages/policies'))
const PolicyDetailsPage = lazy(() => import('./pages/policies/details'))
const PolicyAddOrUpdateForm = lazy(() => import('./pages/policies/form'))
const MfaConfigPage = lazy(() => import('./pages/security/mfa/MfaConfigPage'))
const PasswordPoliciesFormPage = lazy(() => import('./pages/security/password-policies').then(m => ({ default: m.PasswordPoliciesFormPage })))
const SessionManagementFormPage = lazy(() => import('./pages/security/session-management').then(m => ({ default: m.SessionManagementFormPage })))
const TokenConfigPage = lazy(() => import('./pages/security/token/TokenConfigPage'))
const LockoutConfigPage = lazy(() => import('./pages/security/lockout/LockoutConfigPage'))
const RegistrationConfigPage = lazy(() => import('./pages/security/registration/RegistrationConfigPage'))
const ThreatDetectionFormPage = lazy(() => import('./pages/security/threat-detection').then(m => ({ default: m.ThreatDetectionFormPage })))
const TenantSettingsPage = lazy(() => import('./pages/tenant-settings/TenantSettingsPage').then(m => ({ default: m.TenantSettingsPage })))
const SecurityPage = lazy(() => import('./pages/security/SecurityPage'))
const TenantsPage = lazy(() => import('./pages/tenants'))
const TenantDetailsPage = lazy(() => import('./pages/tenants/details'))
const TenantAddOrUpdateForm = lazy(() => import('./pages/tenants/form'))
const RegistrationFlowsPage = lazy(() => import('./pages/registration-flows'))
const RegistrationFlowDetailsPage = lazy(() => import('./pages/registration-flows/details'))
const RegistrationFlowAddOrUpdateForm = lazy(() => import('./pages/registration-flows/form'))
const MonitoringPage = lazy(() => import('./pages/monitoring/MonitoringPage'))
const AuditLogDetailsPage = lazy(() => import('./pages/audit-log/details/AuditLogDetailsPage'))
const AuthEventDetailsPage = lazy(() => import('./pages/log-monitoring/details/AuthEventDetailsPage'))
const BrandingDetailsPage = lazy(() => import('./pages/branding/templates/details/BrandingDetailsPage'))
const BrandingForm = lazy(() => import('./pages/branding/templates/form/BrandingForm'))
const BrandingPage = lazy(() => import('./pages/branding/BrandingPage'))
const EmailTemplateDetailsPage = lazy(() => import('./pages/branding/email-templates/details'))
const EmailTemplateForm = lazy(() => import('./pages/branding/email-templates/form'))
const SmsTemplateDetailsPage = lazy(() => import('./pages/branding/sms-templates/details'))
const SmsTemplateForm = lazy(() => import('./pages/branding/sms-templates/form'))
const EmailConfigPage = lazy(() => import('./pages/messaging/email/EmailConfigPage'))
const SMSConfigPage = lazy(() => import('./pages/messaging/sms/SMSConfigPage'))
const MessagingPage = lazy(() => import('./pages/messaging/MessagingPage'))
const MFAPage = lazy(() => import('./pages/account/MFAPage'))
const MFAIndex = lazy(() => import('./pages/account/MFAPage').then(m => ({ default: m.MFAIndex })))
const TOTPSetupPage = lazy(() => import('./pages/account/TOTPSetupPage'))
const PasskeySetupPage = lazy(() => import('./pages/account/PasskeySetupPage'))
const SMSSetupPage = lazy(() => import('./pages/account/SMSSetupPage'))
const EmailOtpSetupPage = lazy(() => import('./pages/account/EmailOtpSetupPage'))
const ProfilePage = lazy(() => import('./pages/account/ProfilePage'))
const SettingsPage = lazy(() => import('./pages/account/SettingsPage'))
const NotFoundPage = lazy(() => import('./pages/not-found/NotFoundPage'))

function App() {
  // Auth + tenant initialization and all redirect gating now live in
  // AppBootstrap → RouteGuard, which wrap the route tree below.
  return (
    <QueryClientProvider client={queryClient}>
      <StepUpProvider>
      <AppBootstrap>
      <ErrorBoundary>
      <Suspense fallback={<AppLoadingScreen />}>
      <Routes>
        <Route path="/" element={<LoginPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/logout" element={<Navigate to="/login" replace />} />
        <Route path="/no-access" element={<NoAccessPage />} />
        <Route path="/service-unavailable" element={<ServiceUnavailablePage />} />
        <Route path="/auth/callback" element={<OAuthCallbackPage />} />
        <Route path="/setup/tenant" element={<SetupTenantPage />} />
        <Route path="/setup/admin" element={<SetupAdminPage />} />
        <Route element={<PrivateLayout fullWidth />}>
          <Route path="monitoring" element={<MonitoringPage />} />
          <Route path="logs/:eventId" element={<AuthEventDetailsPage />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="security" element={<SecurityPage />} />
          <Route path="security/mfa/configure" element={<MfaConfigPage />} />
          <Route path="security/password/configure" element={<PasswordPoliciesFormPage />} />
          <Route path="security/session/configure" element={<SessionManagementFormPage />} />
          <Route path="security/token/configure" element={<TokenConfigPage />} />
          <Route path="security/lockout/configure" element={<LockoutConfigPage />} />
          <Route path="security/registration/configure" element={<RegistrationConfigPage />} />
          <Route path="security/threat/configure" element={<ThreatDetectionFormPage />} />
          <Route path="settings" element={<TenantSettingsPage />} />
          <Route path="tenants" element={<TenantsPage />} />
          <Route path="tenants/create" element={<TenantAddOrUpdateForm />} />
          <Route path="tenants/:id" element={<TenantDetailsPage />} />
          <Route path="tenants/:id/edit" element={<TenantAddOrUpdateForm />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="users/create" element={<UserAddOrUpdateForm />} />
          <Route path="users/:userId" element={<UserDetailsPage />} />
          <Route path="users/:userId/edit" element={<UserAddOrUpdateForm />} />
          <Route path="invites" element={<InvitationsPage />} />
          <Route path="invites/create" element={<InviteForm />} />
          <Route path="invites/:inviteId" element={<InviteDetailsPage />} />
          <Route path="roles" element={<RolesPage />} />
          <Route path="roles/create" element={<RoleAddOrUpdateForm />} />
          <Route path="roles/:roleId" element={<RoleDetailsPage />} />
          <Route path="roles/:roleId/edit" element={<RoleAddOrUpdateForm />} />

          <Route path="services" element={<ServicesPage />} />
          <Route path="services/create" element={<ServiceAddOrUpdateForm />} />
          <Route path="services/:serviceId" element={<ServiceDetailsPage />} />
          <Route path="services/:serviceId/edit" element={<ServiceAddOrUpdateForm />} />
          <Route path="apis" element={<ApisPage />} />
          <Route path="apis/create" element={<ApiAddOrUpdateForm />} />
          <Route path="apis/:apiId" element={<ApiDetailsPage />} />
          <Route path="apis/:apiId/edit" element={<ApiAddOrUpdateForm />} />
          <Route path="permissions" element={<PermissionsPage />} />
          <Route path="providers/identity" element={<IdentityProvidersPage />} />
          <Route path="providers/identity/create" element={<IdentityProviderAddOrUpdateForm />} />
          <Route path="providers/identity/:providerId" element={<IdentityProviderDetailsPage />} />
          <Route path="providers/identity/:providerId/edit" element={<IdentityProviderAddOrUpdateForm />} />
          <Route path="clients" element={<ClientsPage />} />
          <Route path="clients/create" element={<ClientAddOrUpdateForm />} />
          <Route path="clients/:clientId" element={<ClientDetailsPage />} />
          <Route path="clients/:clientId/edit" element={<ClientAddOrUpdateForm />} />
          <Route path="webhooks/create" element={<WebhookAddOrUpdateForm />} />
          <Route path="webhooks/:webhookId" element={<WebhookDetailsPage />} />
          <Route path="webhooks/:webhookId/edit" element={<WebhookAddOrUpdateForm />} />
          <Route path="events" element={<EventsAndWebhooksPage />} />
          <Route path="audit-log/:uuid" element={<AuditLogDetailsPage />} />
          <Route path="workload-identity" element={<WorkloadIdentityPage />} />
          <Route path="workload-identity/create" element={<WorkloadIdentityAddOrUpdateForm />} />
          <Route path="workload-identity/:federationId" element={<WorkloadIdentityDetailsPage />} />
          <Route path="workload-identity/:federationId/edit" element={<WorkloadIdentityAddOrUpdateForm />} />
          <Route path="policies" element={<PoliciesPage />} />
          <Route path="policies/create" element={<PolicyAddOrUpdateForm />} />
          <Route path="policies/:policyId" element={<PolicyDetailsPage />} />
          <Route path="policies/:policyId/edit" element={<PolicyAddOrUpdateForm />} />
          <Route path="registration-flows" element={<RegistrationFlowsPage />} />
          <Route path="registration-flows/create" element={<RegistrationFlowAddOrUpdateForm />} />
          <Route path="registration-flows/:registrationFlowId" element={<RegistrationFlowDetailsPage />} />
          <Route path="registration-flows/:registrationFlowId/edit" element={<RegistrationFlowAddOrUpdateForm />} />
          <Route path="branding" element={<BrandingPage />} />
          <Route path="branding/templates/create" element={<BrandingForm />} />
          <Route path="branding/templates/:brandingId" element={<BrandingDetailsPage />} />
          <Route path="branding/templates/:brandingId/edit" element={<BrandingForm />} />
          <Route path="branding/email-templates/:templateId" element={<EmailTemplateDetailsPage />} />
          <Route path="branding/email-templates/:templateId/edit" element={<EmailTemplateForm />} />
          <Route path="branding/sms-templates/:templateId" element={<SmsTemplateDetailsPage />} />
          <Route path="branding/sms-templates/:templateId/edit" element={<SmsTemplateForm />} />
          <Route path="messaging" element={<MessagingPage />} />
          <Route path="messaging/email/configure" element={<EmailConfigPage />} />
          <Route path="messaging/sms/configure" element={<SMSConfigPage />} />
          <Route path="account/profile" element={<ProfilePage />} />
          <Route path="account/settings" element={<SettingsPage />} />
          <Route path="account/mfa" element={<MFAPage />}>
            <Route index element={<MFAIndex />} />
            <Route path="totp" element={<TOTPSetupPage />} />
            <Route path="passkeys" element={<PasskeySetupPage />} />
            <Route path="sms" element={<SMSSetupPage />} />
            <Route path="email-otp" element={<EmailOtpSetupPage />} />
          </Route>
          <Route path="*" element={<NotFoundPage />} />
        </Route>
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
