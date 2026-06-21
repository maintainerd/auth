package emailtemplate

const DeviceApprovedEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
{{if .LogoURL}}<img src="{{.LogoURL}}" alt="Logo" style="max-height: 50px; margin: 20px auto;" />{{end}}
<h2>Device Authorization Approved</h2>
<p>You have successfully authorized <strong>{{.ClientName}}</strong> to access your account.</p>
<p>If you did not approve this request, please secure your account immediately.</p>
</body>
</html>`

const DeviceApprovedEmailPlain = `You have successfully authorized {{.ClientName}} to access your account. If you did not approve this request, please secure your account immediately.`
