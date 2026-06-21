package emailtemplate

const CIBANotificationEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; text-align: center;">
{{if .LogoURL}}<img src="{{.LogoURL}}" alt="Logo" style="max-height: 50px; margin: 20px auto;" />{{end}}
<h2>Authentication Request</h2>
<p><strong>{{.ClientName}}</strong> is requesting access to your account.</p>
{{if .BindingMessage}}<p>Message: <strong>{{.BindingMessage}}</strong></p>{{end}}
<p>Please approve or deny this request in your authenticator app or the login portal.</p>
</body>
</html>`

const CIBANotificationEmailPlain = `{{.ClientName}} is requesting access to your account.{{if .BindingMessage}} Message: {{.BindingMessage}}{{end}} Please approve or deny this request in your authenticator app or the login portal.`
