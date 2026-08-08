import LoginLayout from "@/components/layout/LoginLayout"
import ResetPasswordForm from "./components/ResetPasswordForm"
import { useTenant } from '@/hooks/useTenant'

const ResetPasswordPage = () => {
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <ResetPasswordForm />
    </LoginLayout>
  )
}

export default ResetPasswordPage

