import LoginLayout from "@/components/layout/LoginLayout"
import RegisterForm from "./components/RegisterForm"
import { useTenant } from '@/hooks/useTenant'

const RegisterPage = () => {
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <RegisterForm />
    </LoginLayout>
  )
}

export default RegisterPage;
