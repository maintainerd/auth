package emailtemplate

const MagicLinkEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
  <div style="max-width: 480px; margin: auto; background: #fff; padding: 30px; border-radius: 8px; border: 1px solid #e0e0e0;">
    <img src="{{.LogoURL}}" alt="Logo" style="max-width: 150px; margin-bottom: 20px;" />
    <h2>Sign in to your account</h2>
    <div style="font-size: 15px; line-height: 1.6; margin-bottom: 20px;">
      Click the button below to sign in. If you didn't request this, you can safely ignore this email.
    </div>
    <a href="{{.MagicLinkURL}}" style="display: inline-block; margin-top: 20px; padding: 12px 20px; background: #007bff; color: #fff; text-decoration: none; border-radius: 4px;">Sign In</a>
    <div style="font-size: 13px; color: #666; margin-top: 30px; line-height: 1.4;">
      This link will expire in 15 minutes for security reasons. If your link has expired, you can request a new one from the sign-in page.
    </div>
    <div style="font-size: 13px; color: #666; margin-top: 15px; line-height: 1.4;">
      If the button doesn't work, you can copy and paste this link into your browser:<br>
      <a href="{{.MagicLinkURL}}" style="color: #007bff; word-break: break-all;">{{.MagicLinkURL}}</a>
    </div>
  </div>
</body>
</html>`

const MagicLinkEmailPlain = `Sign in to your account

Click the link below to sign in. If you didn't request this, you can safely ignore this email.

{{.MagicLinkURL}}

This link will expire in 15 minutes for security reasons. If your link has expired, you can request a new one from the sign-in page.`
