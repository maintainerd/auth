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

type sendgridProvider struct{ apiKey string }

func newSendGridProvider(cfg ProviderConfig) Provider {
	return &sendgridProvider{apiKey: cfg.APIKey}
}

func (p *sendgridProvider) Send(ctx context.Context, params SendParams) error {
	_, span := otel.Tracer("email").Start(ctx, "sendgrid.send")
	defer span.End()
	span.SetAttributes(attribute.String("email.to", params.To))

	type sgContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type sgAddr struct {
		Email string `json:"email"`
	}
	type sgPersonalisation struct {
		To []sgAddr `json:"to"`
	}
	body := map[string]any{
		"personalizations": []sgPersonalisation{{To: []sgAddr{{Email: params.To}}}},
		"from":             sgAddr{Email: params.From},
		"subject":          params.Subject,
		"content": []sgContent{
			{Type: "text/html", Value: params.BodyHTML},
		},
	}
	if params.BodyPlain != "" {
		body["content"] = []sgContent{
			{Type: "text/plain", Value: params.BodyPlain},
			{Type: "text/html", Value: params.BodyHTML},
		}
	}

	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("sendgrid: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "sendgrid send failed")
		return fmt.Errorf("sendgrid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "sendgrid send failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
