import LoginLayout from "@/components/layout/LoginLayout"
import LoginForm from "./components/LoginForm"
import { useTenant } from '@/hooks/useTenant'

const LoginPage = () => {
  const { currentTenant } = useTenant()

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <LoginForm />
    </LoginLayout>
  )
}

export default LoginPage;
