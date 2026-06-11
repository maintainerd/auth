package email

import (
	"context"
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/gomail.v2"
)

type smtpProvider struct {
	host string
	port int
	user string
	pass string
	from string
}

func newSMTPProvider(cfg ProviderConfig) Provider {
	return &smtpProvider{
		host: cfg.Host,
		port: cfg.Port,
		user: cfg.Username,
		pass: cfg.Password,
		from: cfg.ResolveFrom(""),
	}
}

func (p *smtpProvider) Send(ctx context.Context, params SendParams) error {
	_, span := otel.Tracer("email").Start(ctx, "smtp.send")
	defer span.End()
	span.SetAttributes(
		attribute.String("smtp.host", p.host),
		attribute.Int("smtp.port", p.port),
		attribute.String("email.to", params.To),
		attribute.String("email.subject", params.Subject),
	)

	from := params.From
	if from == "" {
		from = p.from
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", params.To)
	m.SetHeader("Subject", params.Subject)
	if params.BodyPlain != "" {
		m.SetBody("text/plain", params.BodyPlain)
		m.AddAlternative("text/html", params.BodyHTML)
	} else {
		m.SetBody("text/html", params.BodyHTML)
	}

	d := gomail.NewDialer(p.host, p.port, p.user, p.pass)
	d.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: p.host,
	}
	if err := d.DialAndSend(m); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "smtp send failed")
		return fmt.Errorf("smtp: %w", err)
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
