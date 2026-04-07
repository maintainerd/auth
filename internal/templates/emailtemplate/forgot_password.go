package emailtemplate

const ForgotPasswordEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
  <div style="max-width: 480px; margin: auto; background: #fff; padding: 30px; border-radius: 8px; border: 1px solid #e0e0e0;">
    <img src="{{.LogoURL}}" alt="Logo" style="max-width: 150px; margin-bottom: 20px;" />
    <h2>Password Reset Request</h2>
    <div style="font-size: 15px; line-height: 1.6; margin-bottom: 20px;">
      We received a request to reset your password. If you didn't make this request, you can safely ignore this email.
    </div>
    <div style="font-size: 15px; line-height: 1.6; margin-bottom: 20px;">
      To reset your password, click the button below:
    </div>
    <a href="{{.ResetURL}}" style="display: inline-block; margin-top: 20px; padding: 12px 20px; background: #007bff; color: #fff; text-decoration: none; border-radius: 4px;">Reset Password</a>
    <div style="font-size: 13px; color: #666; margin-top: 30px; line-height: 1.4;">
      This link will expire in 1 hour for security reasons. If you need to reset your password after this time, please request a new reset link.
    </div>
    <div style="font-size: 13px; color: #666; margin-top: 15px; line-height: 1.4;">
      If the button doesn't work, you can copy and paste this link into your browser:<br>
      <a href="{{.ResetURL}}" style="color: #007bff; word-break: break-all;">{{.ResetURL}}</a>
    </div>
  </div>
</body>
</html>`

const ForgotPasswordEmailPlain = `Password Reset Request

We received a request to reset your password. If you didn't make this request, you can safely ignore this email.

To reset your password, visit this link:
{{.ResetURL}}

This link will expire in 1 hour for security reasons. If you need to reset your password after this time, please request a new reset link.

If you have any questions, please contact our support team.`
