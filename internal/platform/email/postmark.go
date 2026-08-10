package email

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

type postmarkProvider struct{ serverToken string }

func newPostmarkProvider(cfg ProviderConfig) Provider {
	return &postmarkProvider{serverToken: cfg.APIKey}
}

func (p *postmarkProvider) Send(ctx context.Context, params SendParams) error {
	_, span := otel.Tracer("email").Start(ctx, "postmark.send")
	defer span.End()
	span.SetAttributes(attribute.String("email.to", params.To))

	body := map[string]any{
		"From":     params.From,
		"To":       params.To,
		"Subject":  params.Subject,
		"HtmlBody": params.BodyHTML,
	}
	if params.BodyPlain != "" {
		body["TextBody"] = params.BodyPlain
	}

	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.postmarkapp.com/email", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("postmark: build request: %w", err)
	}
	req.Header.Set("X-Postmark-Server-Token", p.serverToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "postmark send failed")
		return fmt.Errorf("postmark: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("postmark: unexpected status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "postmark send failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
