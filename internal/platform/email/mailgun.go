package email

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type mailgunProvider struct {
	apiKey string
	domain string
}

func newMailgunProvider(cfg ProviderConfig) Provider {
	return &mailgunProvider{apiKey: cfg.APIKey, domain: cfg.Domain}
}

func (p *mailgunProvider) Send(ctx context.Context, params SendParams) error {
	_, span := otel.Tracer("email").Start(ctx, "mailgun.send")
	defer span.End()
	span.SetAttributes(attribute.String("email.to", params.To))

	form := url.Values{}
	form.Set("from", params.From)
	form.Set("to", params.To)
	form.Set("subject", params.Subject)
	form.Set("html", params.BodyHTML)
	if params.BodyPlain != "" {
		form.Set("text", params.BodyPlain)
	}

	endpoint := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", url.PathEscape(p.domain))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("mailgun: build request: %w", err)
	}
	req.SetBasicAuth("api", p.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "mailgun send failed")
		return fmt.Errorf("mailgun: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("mailgun: unexpected status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "mailgun send failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
