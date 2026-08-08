import LoginLayout from "@/components/layout/LoginLayout"
import RegisterInviteForm from "./components/RegisterInviteForm"
import { useTenant } from '@/hooks/useTenant'

const RegisterInvitePage = () => {
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <RegisterInviteForm />
    </LoginLayout>
  )
}

export default RegisterInvitePage
