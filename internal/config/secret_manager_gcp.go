package config

import (
	"context"
	"fmt"
	"strings"

	gcpsm "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

// ────────────────────────────────── GCP Secret Manager provider ────────────
//
// Configuration env vars:
//   GCP_PROJECT_ID – GCP project ID (required)
//
// Authentication: Application Default Credentials (ADC).
// In GKE / Cloud Run, Workload Identity is used automatically.
// Locally, run: gcloud auth application-default login
//
// Secret naming: projects/<GCP_PROJECT_ID>/secrets/<key-lowercased-hyphens>/versions/latest
// e.g. JWT_PRIVATE_KEY → projects/my-project/secrets/jwt-private-key/versions/latest
//
// The SECRET_PREFIX variable is not applied to GCP secret names; use IAM
// policies to scope access instead.

// gcpSMClient abstracts the GCP Secret Manager API for testability.
type gcpSMClient interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
}

// newGCPClient creates a GCP Secret Manager client. Replaceable in tests.
var newGCPClient = func(ctx context.Context) (gcpSMClient, error) {
	return gcpsm.NewClient(ctx)
}

type gcpSecretManager struct {
	projectID string
	client    gcpSMClient
}

func newGCPSecretManager(projectID string) (*gcpSecretManager, error) {
	initCtx, initCancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer initCancel()
	client, err := newGCPClient(initCtx)
	if err != nil {
		return nil, fmt.Errorf("GCP Secret Manager: failed to create client: %w", err)
	}
	return &gcpSecretManager{projectID: projectID, client: client}, nil
}

func (g *gcpSecretManager) secretVersion(key string) string {
	name := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", g.projectID, name)
}

func (g *gcpSecretManager) GetSecret(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()

	version := g.secretVersion(key)
	result, err := g.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: version,
	})
	if err != nil {
		return nil, fmt.Errorf("GCP Secret Manager: failed to access %q: %w", version, err)
	}
	if result.Payload == nil || result.Payload.Data == nil {
		return nil, fmt.Errorf("GCP Secret Manager: secret %q returned nil payload", version)
	}
	return result.Payload.Data, nil
}

func (g *gcpSecretManager) GetSecretString(key string) (string, error) {
	data, err := g.GetSecret(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
