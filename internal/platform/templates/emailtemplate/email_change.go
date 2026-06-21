package emailtemplate

const EmailChangeOTPEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
{{if .LogoURL}}<img src="{{.LogoURL}}" alt="Logo" style="max-height: 50px; margin: 20px auto;" />{{end}}
<h2>Email Change Verification</h2>
<p>Your email change verification code is:</p>
<h1 style="letter-spacing: 8px; font-size: 32px;">{{.OTP}}</h1>
<p>This code expires in 1 hour. If you did not request this change, please ignore this email.</p>
</body>
</html>`

const EmailChangeOTPEmailPlain = `Your email change verification code is: {{.OTP}}. It expires in 1 hour. If you did not request this change, please ignore this email.`
