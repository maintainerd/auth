import LoginLayout from "@/components/layout/LoginLayout"
import RegisterProfileForm from "./components/RegisterProfileForm"
import { useTenant } from '@/hooks/useTenant'

const RegisterProfilePage = () => {
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <RegisterProfileForm />
    </LoginLayout>
  )
}

export default RegisterProfilePage
