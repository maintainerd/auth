package sms

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

type twilioProvider struct {
	accountSID string
	authToken  string
	from       string
}

func newTwilioProvider(cfg ProviderConfig) Provider {
	return &twilioProvider{
		accountSID: cfg.TwilioSID,
		authToken:  cfg.TwilioToken,
		from:       cfg.TwilioFrom,
	}
}

func (p *twilioProvider) Send(ctx context.Context, to, body string) error {
	_, span := otel.Tracer("sms").Start(ctx, "twilio.send")
	defer span.End()
	span.SetAttributes(attribute.String("sms.to", to))

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", url.PathEscape(p.accountSID))

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", p.from)
	form.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("twilio: build request: %w", err)
	}
	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "twilio send failed")
		return fmt.Errorf("twilio: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("twilio: unexpected status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "twilio send failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
