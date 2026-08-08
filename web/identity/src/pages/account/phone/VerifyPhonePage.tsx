import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Phone, ArrowLeft, CheckCircle2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FormInputField } from "@/components/form"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import LoginLayout from "@/components/layout/LoginLayout"
import { useToast } from "@/hooks/useToast"
import { post } from "@/services/api/client"
import { fetchAccount } from "@/services/api/auth"
import { API_ENDPOINTS } from "@/services/api/config"

/**
 * Self-service phone verification. The user sends a one-time code to their phone
 * (POST /account/phone/send-verification) and enters it (POST /account/phone/verify),
 * which sets `phone_verified = true` on their account.
 */
export default function VerifyPhonePage() {
  const queryClient = useQueryClient()
  const { showError, showSuccess } = useToast()
  const [phone, setPhone] = useState("")
  const [code, setCode] = useState("")
  const [codeSent, setCodeSent] = useState(false)

  const { data: account, isLoading } = useQuery({
    queryKey: ["account"],
    queryFn: fetchAccount,
  })

  // Pre-fill the phone field from the account once it loads.
  useEffect(() => {
    if (account?.phone) setPhone(account.phone)
  }, [account?.phone])

  const sendMut = useMutation({
    mutationFn: () => post(API_ENDPOINTS.AUTH.ACCOUNT_PHONE_SEND_VERIFICATION, { phone }),
    onSuccess: () => {
      setCodeSent(true)
      showSuccess("Verification code sent to your phone")
    },
    onError: (e) => showError(e, "Failed to send verification code"),
  })

  const verifyMut = useMutation({
    mutationFn: () => post(API_ENDPOINTS.AUTH.ACCOUNT_PHONE_VERIFY, { phone, code }),
    onSuccess: () => {
      showSuccess("Phone number verified")
      setCodeSent(false)
      setCode("")
      queryClient.invalidateQueries({ queryKey: ["account"] })
    },
    onError: (e) => showError(e, "Failed to verify phone number"),
  })

  const alreadyVerified = account?.phone_verified === true

  return (
    <LoginLayout>
      <div className="container mx-auto max-w-md py-12 px-4">
        <Link
          to="/login-success"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-6"
        >
          <ArrowLeft className="h-3 w-3" /> Back
        </Link>

        <div className="mb-6">
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Phone className="h-6 w-6" /> Verify Phone
          </h1>
          <p className="text-muted-foreground mt-1">
            Confirm your phone number with a one-time code sent by SMS.
          </p>
        </div>

        {isLoading ? (
          <p className="text-muted-foreground">Loading…</p>
        ) : alreadyVerified ? (
          <Card className="border-green-500/40">
            <CardContent className="flex items-center gap-3 py-6" role="status">
              <CheckCircle2 className="h-6 w-6 text-green-600" />
              <div>
                <p className="font-medium">Your phone is verified</p>
                <p className="text-sm text-muted-foreground">{account?.phone}</p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                {codeSent ? "Enter the code we sent" : "Send a verification code"}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormInputField
                id="phone"
                label="Phone number"
                type="tel"
                inputMode="tel"
                autoComplete="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="+15551234567"
                disabled={codeSent || sendMut.isPending}
              />

              {!codeSent ? (
                <Button
                  className="w-full"
                  onClick={() => sendMut.mutate()}
                  disabled={sendMut.isPending || !phone}
                >
                  {sendMut.isPending ? "Sending…" : "Send code"}
                </Button>
              ) : (
                <>
                  <FormInputField
                    id="code"
                    label="Verification code"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    placeholder="123456"
                  />
                  <div className="flex gap-2">
                    <Button
                      className="flex-1"
                      onClick={() => verifyMut.mutate()}
                      disabled={verifyMut.isPending || !code}
                    >
                      {verifyMut.isPending ? "Verifying…" : "Verify"}
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => sendMut.mutate()}
                      disabled={sendMut.isPending}
                    >
                      Resend
                    </Button>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </LoginLayout>
  )
}
