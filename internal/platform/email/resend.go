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

type resendProvider struct{ apiKey string }

func newResendProvider(cfg ProviderConfig) Provider {
	return &resendProvider{apiKey: cfg.APIKey}
}

func (p *resendProvider) Send(ctx context.Context, params SendParams) error {
	_, span := otel.Tracer("email").Start(ctx, "resend.send")
	defer span.End()
	span.SetAttributes(attribute.String("email.to", params.To))

	body := map[string]any{
		"from":    params.From,
		"to":      []string{params.To},
		"subject": params.Subject,
		"html":    params.BodyHTML,
	}
	if params.BodyPlain != "" {
		body["text"] = params.BodyPlain
	}

	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "resend send failed")
		return fmt.Errorf("resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("resend: unexpected status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "resend send failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
