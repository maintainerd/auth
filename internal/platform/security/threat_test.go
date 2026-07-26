package security

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssessLoginThreat_VolumeAloneNeverHardBlocks verifies that raw per-IP
// failure VOLUME — an ambiguous signal a busy shared NAT produces just like an
// attacker — only ever escalates to the step-up band, never a hard block. A
// legitimate user behind a hammered egress IP is asked for a second factor, not
// denied outright.
func TestAssessLoginThreat_VolumeAloneNeverHardBlocks(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	ctx := context.Background()
	const tenantID = int64(1)
	const ip = "203.0.113.5"
	cfg := &ThreatConfig{
		VelocityCheckEnabled:         true,
		RiskBasedStepUpEnabled:       true,
		RiskStepUpThreshold:          21,
		RiskBlockThreshold:           81,
		VelocityFailuresPerIPPerHour: 4,
		DistinctAccountsPerIPPerHour: 10,
	}

	// Four failures, all against the SAME account (distinct fan-out = 1), so only
	// the volume signal is exercised.
	for i := 0; i < 4; i++ {
		RecordLoginThreatFailure(ctx, tenantID, ip, "victim@example.com", cfg)
	}

	d := AssessLoginThreat(ctx, tenantID, ip, "", cfg)
	assert.Equal(t, 60, d.RiskScore, "volume at the limit maps to the step-up band, not block")
	assert.True(t, d.RequiresStepUp, "volume should trigger step-up")
	assert.False(t, d.Blocked, "raw volume alone must never hard-block")
}

// TestAssessLoginThreat_DistinctFanoutBlocksCredentialStuffing verifies that one
// IP failing logins across many DISTINCT accounts — the credential-stuffing
// fingerprint that per-account lockout structurally cannot see — escalates to a
// hard block.
func TestAssessLoginThreat_DistinctFanoutBlocksCredentialStuffing(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	ctx := context.Background()
	const tenantID = int64(1)
	const ip = "198.51.100.7"
	cfg := &ThreatConfig{
		VelocityCheckEnabled:         true,
		RiskBasedStepUpEnabled:       true,
		RiskStepUpThreshold:          21,
		RiskBlockThreshold:           81,
		VelocityFailuresPerIPPerHour: 1000, // keep volume out of the way
		DistinctAccountsPerIPPerHour: 5,
	}

	// 10 distinct accounts from one IP == 2× the distinct limit → block band.
	for i := 0; i < 10; i++ {
		RecordLoginThreatFailure(ctx, tenantID, ip, fmt.Sprintf("user%d@example.com", i), cfg)
	}

	d := AssessLoginThreat(ctx, tenantID, ip, "", cfg)
	assert.Equal(t, 100, d.RiskScore, "aggressive distinct-account fan-out is a hard-block signal")
	assert.True(t, d.Blocked, "credential stuffing must be blocked")
	assert.Contains(t, d.Reason, "credential stuffing")
}

// TestAssessLoginThreat_ModerateDistinctFanoutStepsUp verifies the middle band:
// distinct fan-out at (not above) the limit challenges with step-up rather than
// hard-blocking, so a large legitimate org behind one IP can still prove itself.
func TestAssessLoginThreat_ModerateDistinctFanoutStepsUp(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	ctx := context.Background()
	const tenantID = int64(1)
	const ip = "198.51.100.8"
	cfg := &ThreatConfig{
		VelocityCheckEnabled:         true,
		RiskBasedStepUpEnabled:       true,
		RiskStepUpThreshold:          21,
		RiskBlockThreshold:           81,
		VelocityFailuresPerIPPerHour: 1000,
		DistinctAccountsPerIPPerHour: 5,
	}

	// Exactly the distinct limit (5) → step-up band (75), below block (81).
	for i := 0; i < 5; i++ {
		RecordLoginThreatFailure(ctx, tenantID, ip, fmt.Sprintf("emp%d@example.com", i), cfg)
	}

	d := AssessLoginThreat(ctx, tenantID, ip, "", cfg)
	assert.Equal(t, 75, d.RiskScore)
	assert.True(t, d.RequiresStepUp)
	assert.False(t, d.Blocked, "moderate fan-out challenges, does not lock out")
}

// TestRecordLoginThreatSuccess_DoesNotClearVelocityCounter locks in the fix for
// the per-IP velocity success-clear bypass: a single successful login from an IP
// must NOT reset the aggregate credential-stuffing signal, or an attacker could
// clear it for every account they are stuffing from that host simply by logging
// into one valid account of their own.
func TestRecordLoginThreatSuccess_DoesNotClearVelocityCounter(t *testing.T) {
	saveAndRestoreRateLimiter(t)
	_, cli := newMiniredisClient(t)
	InitRateLimiter(cli)

	ctx := context.Background()
	const tenantID = int64(1)
	const ip = "203.0.113.9"
	cfg := &ThreatConfig{
		VelocityCheckEnabled:         true,
		RiskBasedStepUpEnabled:       true,
		RiskStepUpThreshold:          21,
		RiskBlockThreshold:           81,
		VelocityFailuresPerIPPerHour: 4,
		DistinctAccountsPerIPPerHour: 10,
	}

	for i := 0; i < 4; i++ {
		RecordLoginThreatFailure(ctx, tenantID, ip, "victim@example.com", cfg)
	}
	before := AssessLoginThreat(ctx, tenantID, ip, "", cfg)
	require.Equal(t, 60, before.RiskScore, "velocity should be elevated before success")

	// An attacker logging into their OWN valid account from the same IP must not
	// wipe the aggregate counter.
	RecordLoginThreatSuccess(ctx, tenantID, 42, ip, "test-agent", cfg)

	after := AssessLoginThreat(ctx, tenantID, ip, "", cfg)
	assert.Equal(t, 60, after.RiskScore, "successful login must not reset the per-IP velocity signal")

	got, err := cli.Get(ctx, threatFailureKey(tenantID, ip)).Int()
	require.NoError(t, err, "velocity counter key must still exist after a success")
	assert.Equal(t, 4, got)
}
