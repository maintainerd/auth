package secpolicy

import "github.com/maintainerd/maintainerd-auth/internal/platform/security"

// LoadThreatPolicy returns the effective threat-detection policy for a tenant,
// resolved from security_settings.threat_config with seeded defaults applied.
func LoadThreatPolicy(repo SecuritySettingRepository, tenantID int64) *security.ThreatConfig {
	cfg, _ := DefaultSecuritySettingConfig("threat")
	if repo != nil {
		if ss, err := repo.FindByTenantID(tenantID); err == nil && ss != nil {
			if merged, err := NormalizeSecuritySettingConfig("threat", mapFromJSON(ss.ThreatConfig), nil); err == nil {
				cfg = merged
			}
		}
	}
	return &security.ThreatConfig{
		BruteForceDetectionEnabled:             boolValue(cfg["brute_force_detection_enabled"]),
		ImpossibleTravelDetectionEnabled:       boolValue(cfg["impossible_travel_detection_enabled"]),
		NewDeviceNotificationEnabled:           boolValue(cfg["new_device_notification_enabled"]),
		VelocityCheckEnabled:                   boolValue(cfg["velocity_check_enabled"]),
		RiskBasedStepUpEnabled:                 boolValue(cfg["risk_based_step_up_enabled"]),
		CompromisedCredentialMonitoringEnabled: boolValue(cfg["compromised_credential_monitoring_enabled"]),
		IPReputationCheckEnabled:               boolValue(cfg["ip_reputation_check_enabled"]),
		BlockTorExitNodes:                      boolValue(cfg["block_tor_exit_nodes"]),
		RiskStepUpThreshold:                    intValue(cfg["risk_step_up_threshold"]),
		RiskBlockThreshold:                     intValue(cfg["risk_block_threshold"]),
		VelocityFailuresPerIPPerHour:           intValue(cfg["velocity_failures_per_ip_per_hour"]),
		DistinctAccountsPerIPPerHour:           intValue(cfg["distinct_accounts_per_ip_per_hour"]),
	}
}
