# Threat Detection Configuration

> **Scope**: User Pool · **API Prefix**: `/security-settings/threat` · **Storage**: `security_settings.threat_config` (JSONB)

## Overview

Threat detection configuration controls the system's ability to detect and respond to suspicious authentication patterns in real time. While [lockout configuration](lockout-config.md) handles per-account brute force, threat detection operates at a higher level — analyzing patterns across accounts, IP addresses, geographic locations, and device fingerprints to identify sophisticated attacks.

Threat detection moves security from **reactive** (lock account after N failures) to **proactive** (detect an attack is happening before it succeeds).

---

## Industry Standards & Background

### The Threat Landscape

Authentication systems face a spectrum of attacks with varying sophistication:

| Attack Sophistication | Example | Per-Account Lockout | Threat Detection |
|----------------------|---------|:-------------------:|:----------------:|
| **Low** | Single-account brute force | ✅ Effective | ✅ Detects |
| **Medium** | Credential stuffing (distributed) | ⚠️ May not trigger | ✅ Detects velocity |
| **Medium** | Password spraying | ❌ Below threshold | ✅ Detects cross-account pattern |
| **High** | Impossible travel (stolen tokens) | ❌ No failed auth | ✅ Detects geographic anomaly |
| **High** | Account takeover (correct creds) | ❌ Correct password | ✅ Detects new device/location |
| **Very High** | Compromised credential monitoring | ❌ N/A | ✅ Proactive breach data matching |

### MITRE ATT&CK — Authentication Techniques

| Technique ID | Name | Description | Relevant Detection |
|-------------|------|-------------|-------------------|
| **T1110.001** | Password Guessing | Brute force against single accounts | Lockout + brute force detection |
| **T1110.002** | Password Cracking | Offline cracking of stolen hashes | N/A (server-side) |
| **T1110.003** | Password Spraying | Few passwords against many accounts | Velocity checks, cross-account analysis |
| **T1110.004** | Credential Stuffing | Stolen credential pairs from breaches | Compromised credential monitoring |
| **T1078** | Valid Accounts | Attacker uses legitimate credentials | Impossible travel, new device, risk-based |
| **T1539** | Steal Web Session Cookie | Session hijacking | Impossible travel, device fingerprint |

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-63B §5.2.2** | Throttling Mechanisms | Rate limiting and anti-automation controls |
| **NIST Cybersecurity Framework** | DE.CM — Security Continuous Monitoring | Continuously monitor for anomalous activity |
| **NIST Cybersecurity Framework** | DE.AE — Anomalies and Events | Detect and analyze anomalies |
| **OWASP ASVS v4** | V2.2.7 | "Verify that additional authentication controls are implemented for high-risk transactions" |
| **OWASP** | Credential Stuffing Prevention | Multi-layered defense: rate limiting, credential breach monitoring, device fingerprinting |
| **PCI DSS v4.0 Req 10** | Log and Monitor | Detect and respond to anomalies in access to cardholder data |
| **MITRE ATT&CK** | T1110 | Brute Force — map detection capabilities to attack techniques |
| **ISO 27001:2022** | A.8.16 — Monitoring Activities | Monitor for indicators of compromise |

### Brute Force Detection — Beyond Account Lockout

Per-account lockout has blind spots. System-wide brute force detection adds:

| Dimension | Account Lockout | Brute Force Detection |
|-----------|:--------------:|:--------------------:|
| Per-account attempts | ✅ | ✅ |
| Per-IP attempts (across accounts) | ❌ | ✅ |
| Velocity anomalies (10x normal rate) | ❌ | ✅ |
| Distributed attacks (botnets) | ❌ | ✅ (IP correlation) |
| Geographic correlation | ❌ | ✅ |
| Time-of-day anomalies | ❌ | ✅ |

### Impossible Travel Detection

Impossible travel flags logins that cannot be physically possible:

```
Login 1: New York, USA at 14:00 UTC
Login 2: Tokyo, Japan at 14:30 UTC

Distance: 10,800 km
Time difference: 30 minutes
Required speed: 21,600 km/h (commercial flight ≈ 900 km/h)

Result: IMPOSSIBLE TRAVEL DETECTED
```

**Implementation considerations:**
- GeoIP databases (MaxMind, IP2Location) for IP-to-location mapping
- Calculate distance between locations using Haversine formula
- Define maximum travel speed (typically 1,000 km/h for commercial aviation)
- Account for VPN usage (flag but don't block — many legitimate users use VPNs)
- Account for mobile IP changes (cellular to WiFi — same city)
- Accuracy depends on GeoIP precision (city-level vs. country-level)

### New Device Detection

Track known devices per user and flag unknown devices:

```
Known Devices for user@example.com:
1. Chrome 120 / macOS / San Francisco (last seen: today)
2. Safari / iOS 17 / San Francisco (last seen: yesterday)

New Login:
Firefox 119 / Windows 11 / Moscow
→ NEW DEVICE + NEW LOCATION → High Risk
```

**Device fingerprinting factors:**
- User-agent string (browser, OS, version)
- Screen resolution
- Timezone
- Language preferences
- Installed plugins/fonts (with privacy considerations)
- TLS fingerprint (JA3/JA4)
- Canvas/WebGL fingerprint (with privacy considerations)

### Velocity Checks

Velocity checks detect abnormal patterns that don't trigger per-account lockout:

| Pattern | Normal | Anomalous |
|---------|--------|-----------|
| Failed logins per IP per hour | 1-5 | 50+ |
| New account registrations per IP per hour | 0-2 | 20+ |
| Password resets per IP per hour | 0-1 | 10+ |
| Successful logins per IP per minute | 0-2 | 10+ (credential stuffing with valid creds) |
| Different accounts from same IP per hour | 1-3 | 50+ |

### Risk-Based Step-Up Authentication

Instead of binary allow/deny, risk-based authentication adapts the authentication requirements:

```
Risk Score Calculation:
  Base Score = 0 (trusted)
  
  + 20: New device
  + 20: New location (city)
  + 40: New country
  + 60: Impossible travel
  + 30: Multiple failed attempts recently
  + 20: Unusual time of day
  + 50: IP from known malicious list
  + 30: Using Tor/VPN exit node
  + 40: Credential found in breach database
  
  TOTAL → Risk Level → Action

  0-20:  LOW      → Normal login
  21-50: MEDIUM   → Require MFA (even if normally optional)
  51-80: HIGH     → Require MFA + email confirmation
  81+:   CRITICAL → Block login, notify admin, email user
```

### Compromised Credential Monitoring

Check user credentials against known breach databases:
- **At registration**: Reject passwords found in breach data (NIST recommendation)
- **At login**: Flag/require password change if current password is in breaches
- **Proactively**: Batch check credentials against newly released breach data
- **Sources**: HaveIBeenPwned API (k-anonymity model), internal breach databases, commercial feeds

---

## Our Implementation

### Data Model

Stored in `security_settings.threat_config` (JSONB column). The config is schema-free (`map[string]any`).

**Expected JSONB fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `brute_force_detection_enabled` | bool | false | Enable system-wide brute force pattern detection |
| `impossible_travel_detection_enabled` | bool | false | Flag logins from geographically impossible locations |
| `new_device_notification_enabled` | bool | false | Notify users when a new device logs into their account |
| `velocity_check_enabled` | bool | false | Detect abnormal authentication velocity patterns |
| `risk_based_step_up_enabled` | bool | false | Adaptive authentication based on risk scoring |
| `compromised_credential_monitoring_enabled` | bool | false | Check credentials against known breach databases |

### API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/security-settings/threat` | `GetThreatConfig` | Returns the current threat detection configuration |
| `PUT` | `/security-settings/threat` | `UpdateThreatConfig` | Replaces the threat detection configuration |

**Example request body (PUT):**
```json
{
  "brute_force_detection_enabled": true,
  "impossible_travel_detection_enabled": true,
  "new_device_notification_enabled": true,
  "velocity_check_enabled": true,
  "risk_based_step_up_enabled": false,
  "compromised_credential_monitoring_enabled": true
}
```

### Service Layer

- **`GetThreatConfig(ctx, userPoolID)`** — Lazy-creates the security setting row, then returns the `threat_config` JSONB.
- **`UpdateThreatConfig(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)`** — Calls `updateConfig` with `"threat"` config type.

The update runs in a transaction:
1. Find or create `security_settings` row for the user pool
2. Replace the `threat_config` JSONB column
3. Increment `version`
4. Create a `security_settings_audit` row with `change_type: "update_threat_config"`

### Audit Trail

Every update creates a `security_settings_audit` row:
- `change_type`: `"update_threat_config"`
- `old_config`: Previous threat JSONB
- `new_config`: New threat JSONB
- `ip_address`, `user_agent`, `created_by`: Admin context

### Validation

Currently minimal — only validates the config map is non-empty. No schema validation.

**Source files:**
- Model: `internal/model/security_setting.go`
- Service: `internal/service/security_setting.go`
- Handler: `internal/rest/security_setting_handler.go`
- Routes: `internal/rest/security_setting_routes.go`
- DTO: `internal/dto/security_setting.go`

---

## Requirements Checklist

### Configuration Management (Admin API)
- [x] Get threat config via `GET /security-settings/threat`
- [x] Update threat config via `PUT /security-settings/threat`
- [x] Stored as JSONB for flexible schema evolution
- [x] Version tracking (auto-incremented on each update)
- [x] Audit trail (old/new config, who changed, IP, user agent)
- [x] Lazy-create on first access
- [x] OpenTelemetry span tracing
- [x] Unit tests for service layer
- [x] Unit tests for handler layer
- [ ] Sane defaults on creation (all features disabled)

### Brute Force Detection
- [ ] Track failed login attempts per IP address (not just per account)
- [ ] Detect high failure rates across multiple accounts from single IP
- [ ] Configurable threshold (e.g., 50 failures per IP per hour)
- [ ] Response: block IP temporarily, require CAPTCHA, or alert admin
- [ ] Integrate with account lockout for coordinated defense
- [ ] Differentiate between brute force and credential stuffing patterns
- [ ] Real-time detection (not batch — must catch during attack)
- [ ] Store blocked IPs in Redis for performance
- [ ] Auto-expire IP blocks after configurable duration

### Impossible Travel Detection
- [ ] GeoIP lookup on every authentication (MaxMind or equivalent)
- [ ] Store last known login location per user
- [ ] Calculate distance between consecutive logins (Haversine formula)
- [ ] Flag when required travel speed exceeds threshold (1,000 km/h)
- [ ] Account for known VPN usage (reduce false positives)
- [ ] Account for mobile IP switching (cellular ↔ WiFi)
- [ ] Configurable response: log, notify, require MFA, block
- [ ] GeoIP database auto-update (MaxMind releases monthly)

### New Device Detection
- [ ] Fingerprint devices on login (user-agent, screen, timezone, etc.)
- [ ] Store known devices per user
- [ ] Detect unknown device on login
- [ ] Send notification email to user on new device login
- [ ] Include device details in notification (browser, OS, location, time)
- [ ] User can view their known devices
- [ ] User can remove known devices
- [ ] Admin can view devices for any user

### Velocity Checks
- [ ] Monitor authentication attempt rate per IP per time window
- [ ] Monitor authentication attempt rate per account per time window
- [ ] Monitor registration rate per IP per time window
- [ ] Monitor password reset rate per IP per time window
- [ ] Configurable thresholds per action type
- [ ] Response: CAPTCHA, temporary block, alert
- [ ] Sliding window counters (Redis-based for performance)

### Risk-Based Step-Up Authentication
- [ ] Risk scoring engine evaluating multiple signals
- [ ] Signals: new device, new location, impossible travel, failed attempts, time of day, IP reputation
- [ ] Configurable risk thresholds (low/medium/high/critical)
- [ ] Configurable responses per risk level:
  - Low: normal login
  - Medium: require MFA
  - High: require MFA + email confirmation
  - Critical: block + notify admin
- [ ] Risk score included in authentication event log
- [ ] Admin can view risk scores for recent logins
- [ ] Machine learning pipeline for risk model improvement (future)

### Compromised Credential Monitoring
- [ ] Check passwords against breach database at registration
- [ ] Check passwords against breach database at login
- [ ] Use k-anonymity model (HaveIBeenPwned API) — never send full password hash
- [ ] Flag accounts with compromised credentials
- [ ] Force password change for compromised credentials
- [ ] Proactive batch checking against new breach data releases
- [ ] Configurable response: warn, force change, block

### Logging & Alerting
- [ ] Log all detected threats with full context (IP, user, type, risk score)
- [ ] Threat events feed into OpenTelemetry traces and metrics
- [ ] Alert admin on high/critical risk events
- [ ] Alert admin on sustained brute force attacks
- [ ] Dashboard metrics: threat events by type, by time, by severity
- [ ] Webhook event for threat detection (integrate with SIEM)

### IP Reputation
- [ ] Maintain IP reputation scores based on historical behavior
- [ ] Integrate with external IP reputation feeds (optional)
- [ ] Flag Tor exit nodes and known proxy/VPN IPs
- [ ] Flag IPs from hosting providers (cloud VMs often used for attacks)
- [ ] IP allowlist/blocklist for known-good/known-bad IPs (see [IP Restriction Rules](ip-restriction-rules.md))

### Integration & Testing
- [ ] Unit tests for threat config service methods
- [ ] Integration test: brute force detection triggers on high failure rate
- [ ] Integration test: impossible travel detection with mocked GeoIP
- [ ] Integration test: new device notification email
- [ ] Integration test: velocity check blocking
- [ ] Integration test: risk score calculation
- [ ] Integration test: audit trail with threat config changes
- [ ] Load test: threat detection under high traffic (latency impact)

---

## References

- [NIST SP 800-63B §5.2.2 — Throttling Mechanisms](https://pages.nist.gov/800-63-3/sp800-63b.html#throttle)
- [NIST Cybersecurity Framework — Detect Function](https://www.nist.gov/cyberframework)
- [OWASP Credential Stuffing Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Credential_Stuffing_Prevention_Cheat_Sheet.html)
- [OWASP ASVS v4.0 — V2.2 General Authenticator Security](https://owasp.org/www-project-application-security-verification-standard/)
- [MITRE ATT&CK — T1110 Brute Force](https://attack.mitre.org/techniques/T1110/)
- [MITRE ATT&CK — T1078 Valid Accounts](https://attack.mitre.org/techniques/T1078/)
- [MaxMind GeoIP Databases](https://www.maxmind.com/en/geoip-databases)
- [HaveIBeenPwned API — k-Anonymity Model](https://haveibeenpwned.com/API/v3#SearchingPwnedPasswordsByRange)
- [PCI DSS v4.0 Requirement 10 — Log and Monitor](https://www.pcisecuritystandards.org/)
