package emailtemplate

const MFAEnrollEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
{{if .LogoURL}}<img src="{{.LogoURL}}" alt="Logo" style="max-height: 50px; margin: 20px auto;" />{{end}}
<h2>MFA Enrollment Verification</h2>
<p>Your MFA enrollment verification code is:</p>
<h1 style="letter-spacing: 8px; font-size: 32px;">{{.OTP}}</h1>
<p>Enter this code to complete your email OTP MFA enrollment. This code expires in 5 minutes.</p>
</body>
</html>`

const MFAEnrollEmailPlain = `Your MFA enrollment verification code is: {{.OTP}}. Enter this code to complete your email OTP MFA enrollment. This code expires in 5 minutes.`

const MFAStepUpEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
{{if .LogoURL}}<img src="{{.LogoURL}}" alt="Logo" style="max-height: 50px; margin: 20px auto;" />{{end}}
<h2>Verification Code</h2>
<p>Your verification code is:</p>
<h1 style="letter-spacing: 8px; font-size: 32px;">{{.OTP}}</h1>
<p>Use this code to verify your identity. This code expires in 5 minutes.</p>
</body>
</html>`

const MFAStepUpEmailPlain = `Your verification code is: {{.OTP}}. Use this code to verify your identity. This code expires in 5 minutes.`
