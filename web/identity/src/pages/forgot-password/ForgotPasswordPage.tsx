import LoginLayout from "@/components/layout/LoginLayout"
import ForgotPasswordForm from "./components/ForgotPasswordForm"
import { useTenant } from '@/hooks/useTenant'

const ForgotPasswordPage = () => {
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <ForgotPasswordForm />
    </LoginLayout>
  )
}

export default ForgotPasswordPage

