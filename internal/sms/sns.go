package sms

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type snsProvider struct{ client *sns.Client }

func newSNSProvider(ctx context.Context, cfg ProviderConfig) (Provider, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.SNSRegion))
	if err != nil {
		return nil, fmt.Errorf("sms/sns: load aws config: %w", err)
	}
	return &snsProvider{client: sns.NewFromConfig(awsCfg)}, nil
}

func (p *snsProvider) Send(ctx context.Context, to, body string) error {
	_, span := otel.Tracer("sms").Start(ctx, "sns.send")
	defer span.End()
	span.SetAttributes(attribute.String("sms.to", to))

	_, err := p.client.Publish(ctx, &sns.PublishInput{
		PhoneNumber: aws.String(to),
		Message:     aws.String(body),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "sns send failed")
		return fmt.Errorf("sms/sns: %w", err)
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
