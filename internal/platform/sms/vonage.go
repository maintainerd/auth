package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type vonageProvider struct {
	apiKey    string
	apiSecret string
	from      string
}

func newVonageProvider(cfg ProviderConfig) Provider {
	return &vonageProvider{
		apiKey:    cfg.VonageAPIKey,
		apiSecret: cfg.VonageAPISecret,
		from:      cfg.VonageFrom,
	}
}

func (p *vonageProvider) Send(ctx context.Context, to, body string) error {
	_, span := otel.Tracer("sms").Start(ctx, "vonage.send")
	defer span.End()
	span.SetAttributes(attribute.String("sms.to", to))

	payload := map[string]string{
		"api_key":    p.apiKey,
		"api_secret": p.apiSecret,
		"to":         to,
		"from":       p.from,
		"text":       body,
	}
	raw, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://rest.nexmo.com/sms/json", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("vonage: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "vonage send failed")
		return fmt.Errorf("vonage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("vonage: unexpected status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "vonage send failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
