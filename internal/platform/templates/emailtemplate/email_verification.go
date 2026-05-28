package emailtemplate

const EmailVerificationEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
  <div style="max-width: 480px; margin: auto; background: #fff; padding: 30px; border-radius: 8px; border: 1px solid #e0e0e0;">
    <img src="{{.LogoURL}}" alt="Logo" style="max-width: 150px; margin-bottom: 20px;" />
    <h2>Verify Your Email Address</h2>
    <div style="font-size: 15px; line-height: 1.6; margin-bottom: 20px;">
      Use the verification code below to confirm your email address. If you didn't create an account, you can safely ignore this email.
    </div>
    <div style="display: inline-block; margin: 20px 0; padding: 16px 28px; background: #f4f6f8; border: 1px solid #e0e0e0; border-radius: 6px; font-family: 'Courier New', monospace; font-size: 28px; letter-spacing: 8px; font-weight: bold; color: #1a1a1a;">{{.OTP}}</div>
    <div style="font-size: 13px; color: #666; margin-top: 20px; line-height: 1.4;">
      This code will expire in 1 hour for security reasons. If your code has expired, you can request a new one from the sign-in page.
    </div>
  </div>
</body>
</html>`

const EmailVerificationEmailPlain = `Verify Your Email Address

Use the verification code below to confirm your email address. If you didn't create an account, you can safely ignore this email.

Verification code: {{.OTP}}

This code will expire in 1 hour for security reasons. If your code has expired, you can request a new one from the sign-in page.`
