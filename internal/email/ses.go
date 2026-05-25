package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type sesProvider struct {
	client *sesv2.Client
}

func newSESProvider(ctx context.Context, cfg ProviderConfig) (Provider, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("ses: load aws config: %w", err)
	}
	return &sesProvider{client: sesv2.NewFromConfig(awsCfg)}, nil
}

func (p *sesProvider) Send(ctx context.Context, params SendParams) error {
	_, span := otel.Tracer("email").Start(ctx, "ses.send")
	defer span.End()
	span.SetAttributes(
		attribute.String("email.to", params.To),
		attribute.String("email.subject", params.Subject),
	)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(params.From),
		Destination: &types.Destination{
			ToAddresses: []string{params.To},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(params.Subject)},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(params.BodyHTML)},
				},
			},
		},
	}
	if params.BodyPlain != "" {
		input.Content.Simple.Body.Text = &types.Content{Data: aws.String(params.BodyPlain)}
	}

	if _, err := p.client.SendEmail(ctx, input); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ses send failed")
		return fmt.Errorf("ses: %w", err)
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
