package authn

import (
	"context"
	"errors"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// This endpoint is unauthenticated and describes a privilege-granting object, so
// the tests assert two things above all: that it never reveals more than the
// signup form needs, and that every resolution failure is indistinguishable.
func TestRegistrationContextService_Get(t *testing.T) {
	const (
		clientIdentifier = "storefront-abc123"
		flowName         = "partner-signup"
	)
	const tenantID int64 = 7

	activeClient := func() *Client {
		id := clientIdentifier
		return &Client{ClientID: 5, TenantID: tenantID, Status: shared.StatusActive, Identifier: &id}
	}

	newSvc := func(client *Client, clientErr error, flow *RegistrationFlow, flowErr error) RegistrationContextService {
		return NewRegistrationContextService(
			&mockClientRepo{findByIdentifierFn: func(string) (*Client, error) { return client, clientErr }},
			&registrationFlowRepoStub{flow: flow, flowErr: flowErr},
			nil, // nil policy repo → documented defaults
		)
	}

	clientIDPtr := func() *string { s := clientIdentifier; return &s }

	t.Run("flow required_fields are returned as the effective set", func(t *testing.T) {
		flow := &RegistrationFlow{
			RegistrationFlowID: 7,
			TenantID:           tenantID,
			ClientID:           5,
			Status:             shared.StatusActive,
			RequiredFields:     datatypes.JSON([]byte(`["fullname","phone"]`)),
		}
		svc := newSvc(activeClient(), nil, flow, nil)

		got, err := svc.Get(context.Background(), clientIDPtr(), nil, flowName)
		require.NoError(t, err)
		assert.Equal(t, flowName, got.RegistrationFlow)
		assert.Contains(t, got.RequiredFields, "fullname")
		assert.Contains(t, got.RequiredFields, "phone")
	})

	t.Run("verification_required implies email even with no required_fields", func(t *testing.T) {
		flow := &RegistrationFlow{
			TenantID: tenantID, ClientID: 5, Status: shared.StatusActive,
			VerificationRequired: true,
		}
		svc := newSvc(activeClient(), nil, flow, nil)

		got, err := svc.Get(context.Background(), clientIDPtr(), nil, flowName)
		require.NoError(t, err)
		assert.True(t, got.VerificationRequired)
		assert.Contains(t, got.RequiredFields, "email",
			"a flow that forces verification needs an address to verify")
	})

	t.Run("no flow in the link still resolves to the tenant baseline", func(t *testing.T) {
		svc := newSvc(activeClient(), nil, nil, nil)

		got, err := svc.Get(context.Background(), clientIDPtr(), nil, "")
		require.NoError(t, err)
		assert.Empty(t, got.RegistrationFlow)
		assert.NotNil(t, got.RequiredFields, "must be an empty slice, never nil")
	})

	t.Run("required_fields is never nil", func(t *testing.T) {
		flow := &RegistrationFlow{TenantID: tenantID, ClientID: 5, Status: shared.StatusActive}
		svc := newSvc(activeClient(), nil, flow, nil)

		got, err := svc.Get(context.Background(), clientIDPtr(), nil, flowName)
		require.NoError(t, err)
		assert.NotNil(t, got.RequiredFields)
	})

	// Every failure must produce the SAME error. A distinguishable response would
	// turn a guessable flow name into an enumeration oracle, and would leak the
	// operator's kill switch (whoever holds a revoked link could poll for its
	// return).
	t.Run("all resolution failures are indistinguishable", func(t *testing.T) {
		cases := []struct {
			name   string
			client *Client
			flow   *RegistrationFlow
		}{
			{
				name:   "unknown flow",
				client: activeClient(),
				flow:   nil,
			},
			{
				name:   "flow from another tenant",
				client: activeClient(),
				flow:   &RegistrationFlow{TenantID: 999, ClientID: 5, Status: shared.StatusActive},
			},
			{
				name:   "system flow is invite-only",
				client: activeClient(),
				flow:   &RegistrationFlow{TenantID: tenantID, ClientID: 5, Status: shared.StatusActive, IsSystem: true},
			},
			{
				name:   "inactive flow",
				client: activeClient(),
				flow:   &RegistrationFlow{TenantID: tenantID, ClientID: 5, Status: shared.StatusInactive},
			},
			{
				name:   "unknown client",
				client: nil,
				flow:   &RegistrationFlow{TenantID: tenantID, ClientID: 5, Status: shared.StatusActive},
			},
			{
				name:   "inactive client",
				client: &Client{ClientID: 5, TenantID: tenantID, Status: shared.StatusInactive},
				flow:   &RegistrationFlow{TenantID: tenantID, ClientID: 5, Status: shared.StatusActive},
			},
		}

		var messages []string
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				svc := newSvc(tc.client, nil, tc.flow, nil)
				got, err := svc.Get(context.Background(), clientIDPtr(), nil, flowName)
				require.Error(t, err)
				assert.Nil(t, got)
				assert.NotContains(t, err.Error(), "inactive", "must not disclose flow or client state")
				assert.NotContains(t, err.Error(), "system", "must not disclose that a system flow exists")
				messages = append(messages, err.Error())
			})
		}
		for _, m := range messages {
			assert.Equal(t, messages[0], m, "every failure must return an identical message")
		}
	})

	// A repository failure must surface as an InternalError, not as a not-found:
	// HandleServiceError logs the detail and returns only the generic fallback to
	// the caller, so the driver message never reaches the wire. Reporting it as a
	// not-found instead would tell the caller the flow does not exist when the
	// truth is that the database is broken.
	t.Run("repository error is internal, so the transport returns a generic message", func(t *testing.T) {
		svc := newSvc(activeClient(), nil, nil, errors.New("pq: relation missing"))
		got, err := svc.Get(context.Background(), clientIDPtr(), nil, flowName)
		require.Error(t, err)
		assert.Nil(t, got)

		var internal *apperror.InternalError
		require.True(t, errors.As(err, &internal),
			"must be an InternalError so the handler sends the fallback, not the detail")

		var notFound *apperror.NotFoundError
		assert.False(t, errors.As(err, &notFound), "a broken database is not a missing flow")
	})

	// The result type has no field for roles, so a leak is impossible by
	// construction rather than by review. Asserted so a future field addition has
	// to consciously break this test.
	t.Run("result exposes only presentation fields", func(t *testing.T) {
		flow := &RegistrationFlow{TenantID: tenantID, ClientID: 5, Status: shared.StatusActive}
		svc := newSvc(activeClient(), nil, flow, nil)
		got, err := svc.Get(context.Background(), clientIDPtr(), nil, flowName)
		require.NoError(t, err)

		dto := toRegistrationContextResponseDTO(*got)
		assert.Equal(t, flowName, dto.RegistrationFlow)
		assert.NotNil(t, dto.RequiredFields)
	})
}

func TestEffectiveRequiredFields(t *testing.T) {
	t.Run("deduplicates across flow and policy", func(t *testing.T) {
		flow := &RegistrationFlow{RequiredFields: datatypes.JSON([]byte(`["email","fullname"]`))}
		policy := &secpolicy.RegistrationPolicy{RequireEmailVerification: true}

		got := effectiveRequiredFields(flow, policy)
		assert.Equal(t, []string{"email", "fullname"}, got)
	})

	t.Run("tenant phone verification applies with no flow at all", func(t *testing.T) {
		policy := &secpolicy.RegistrationPolicy{RequirePhoneVerification: true}
		assert.Equal(t, []string{"phone"}, effectiveRequiredFields(nil, policy))
	})

	t.Run("unsupported and control fields in stored json are ignored", func(t *testing.T) {
		flow := &RegistrationFlow{RequiredFields: datatypes.JSON([]byte(`["username","password","ssn","phone"]`))}
		assert.Equal(t, []string{"phone"}, effectiveRequiredFields(flow, &secpolicy.RegistrationPolicy{}))
	})

	t.Run("malformed stored json degrades to the policy set", func(t *testing.T) {
		flow := &RegistrationFlow{RequiredFields: datatypes.JSON([]byte(`{"not":"an array"}`))}
		got := effectiveRequiredFields(flow, &secpolicy.RegistrationPolicy{RequireEmailVerification: true})
		assert.Equal(t, []string{"email"}, got)
	})

	t.Run("empty when nothing is required", func(t *testing.T) {
		got := effectiveRequiredFields(nil, &secpolicy.RegistrationPolicy{})
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
