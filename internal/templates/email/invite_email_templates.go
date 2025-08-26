package emailtemplate

const InviteEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
  <div style="max-width: 480px; margin: auto; background: #fff; padding: 30px; border-radius: 8px; border: 1px solid #e0e0e0;">
    <img src="{{.LogoURL}}" alt="Logo" style="max-width: 150px; margin-bottom: 20px;" />
    <h2>You're Invited!</h2>
    <div style="font-size: 15px; line-height: 1.6;">
      <strong>Admin</strong> has invited you to join our organization.
    </div>
    <a href="{{.InviteURL}}" style="display: inline-block; margin-top: 20px; padding: 12px 20px; background: #007bff; color: #fff; text-decoration: none; border-radius: 4px;">Accept Invitation</a>
    <p style="font-size: 12px; color: #999; margin-top: 30px;">If you did not expect this email, you can safely ignore it.</p>
  </div>
</body>
</html>`
