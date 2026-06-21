# SOC 2 Readiness Checklist

This checklist uses the 80 public controls listed in the
[SuperTokens Trust Center](https://security.supertokens.com/controls) as a
readiness target for Maintainerd Auth. The source list was captured on
2026-06-21.

This is an internal gap tracker, not a claim that Maintainerd Auth or its
operator is SOC 2 certified. Application code can satisfy only part of SOC 2;
many controls require company policies, people, infrastructure configuration,
recurring reviews, and retained audit evidence.

## How to use this checklist

- `[x]` means the repository currently contains implementation or maintained
  documentation that directly supports the control.
- `[ ]` with **Partial** means useful implementation exists, but the full
  control or recurring evidence is not present.
- `[ ]` with **External evidence required** means the control cannot be proven
  from this application repository.
- A checked item still needs dated operating evidence during an audit period.
- Add an owner, review date, and evidence link before treating an item as
  audit-ready.

Current repository review: **13 of 80 controls have direct app-level evidence**.

| Control family | Evidenced | Total |
|---|---:|---:|
| Infrastructure security | 2 | 21 |
| Organizational security | 1 | 14 |
| Product security | 3 | 5 |
| Internal security procedures | 5 | 37 |
| Data and privacy | 2 | 3 |
| **Total** | **13** | **80** |

## Infrastructure security

- [ ] Unique production database authentication enforced — **External evidence required:** verify unique production credentials or SSH identities in the deployed environment.
- [ ] Unique account authentication enforced — **External evidence required:** verify unique identities for workforce systems and production access.
- [ ] Production application access restricted — **Partial:** the app provides JWT authentication, RBAC, permissions, and tenant isolation; production workforce access evidence is still required. See `internal/platform/middleware/jwt_middleware.go`, `internal/platform/middleware/permission_middleware.go`, and `internal/iam/`.
- [ ] Access control procedures established — **External evidence required:** document joiner, mover, and leaver procedures and retain completed access requests.
- [ ] Production database access restricted — **External evidence required:** supply database IAM, role, and access-review evidence from the production platform.
- [ ] Firewall access restricted — **External evidence required:** supply firewall administration roles and access-review evidence.
- [ ] Production OS access restricted — **External evidence required:** supply host or cluster administration roles and access-review evidence.
- [ ] Production network access restricted — **External evidence required:** supply production network IAM and access-review evidence.
- [ ] Access revoked upon termination — **External evidence required:** document termination SLAs and retain deprovisioning evidence.
- [ ] Unique network system authentication enforced — **External evidence required:** verify unique production network identities and approved authentication mechanisms.
- [ ] Remote access MFA enforced — **External evidence required:** the product supports MFA, but workforce access to production must separately enforce it. See `internal/mfa/`.
- [ ] Remote access encrypted enforced — **External evidence required:** verify VPN, zero-trust, or equivalent encrypted administrative access.
- [ ] Intrusion detection system utilized — **External evidence required:** deploy and evidence network or cloud intrusion detection for production.
- [x] Log management utilized — structured logs, trace correlation, security events, and configuration audit records are documented in `docs/documentations/settings/logging-and-audit.md`; implementation lives in `internal/authevent/`, `internal/secpolicy/model_settings_audit.go`, and `internal/platform/middleware/logging_middleware.go`.
- [x] Infrastructure performance monitored — OpenTelemetry, Prometheus metrics, health checks, and Grafana provisioning are present in `internal/platform/telemetry/`, `deploy/prometheus/`, `deploy/grafana/`, `docker-compose.yml`, and `docs/documentations/observability/opentelemetry.md`.
- [ ] Network segmentation implemented — **External evidence required:** local Compose networking is not proof of production segmentation.
- [ ] Network firewalls reviewed — **External evidence required:** perform at least annual firewall-rule reviews and retain remediation records.
- [ ] Network firewalls utilized — **External evidence required:** provide production firewall, security-group, ingress, or network-policy configuration.
- [ ] Network and system hardening standards maintained — **Partial:** secure production settings are described in `SECURITY.md` and `docs/documentations/devops/operator-runbook.md`; a versioned hardening standard and annual review evidence are still needed.
- [ ] Service infrastructure maintained — **Partial:** daily SAST, dependency, and secret scanning exists in `.github/workflows/security.yml`; production patching and remediation SLAs are not evidenced here.
- [ ] Encryption key access restricted — **Partial:** pluggable secret-manager support and production secret guidance are documented in `SECURITY.md`; production key IAM and access reviews must be evidenced externally.

## Organizational security

- [ ] Asset disposal procedures utilized — **External evidence required:** establish media sanitization and certificates-of-destruction procedures.
- [ ] Production inventory maintained — **External evidence required:** maintain a production asset and service inventory outside this repository.
- [ ] Portable media encrypted — **External evidence required:** enforce and evidence removable-media encryption or a prohibition on removable media.
- [ ] Anti-malware technology utilized — **External evidence required:** provide endpoint or workload anti-malware coverage and update evidence.
- [ ] Employee background checks performed — **External evidence required:** retain lawful pre-employment screening evidence.
- [ ] Code of Conduct acknowledged by contractors — **External evidence required:** retain contractor acknowledgements.
- [ ] Code of Conduct acknowledged by employees and enforced — **External evidence required:** retain employee acknowledgements and disciplinary procedures.
- [ ] Confidentiality Agreement acknowledged by contractors — **External evidence required:** retain signed contractor confidentiality agreements.
- [ ] Confidentiality Agreement acknowledged by employees — **External evidence required:** retain signed employee confidentiality agreements.
- [ ] Performance evaluations conducted — **External evidence required:** retain annual performance-review evidence.
- [x] Password policy enforced — tenant-configurable length, complexity, history, breached-password, and lockout controls are implemented in `internal/platform/security/password_policy.go`, `internal/authn/service_password_policy.go`, and `internal/secpolicy/password_policy.go`, with configuration documented under `docs/documentations/settings/security-settings/`.
- [ ] MDM system utilized — **External evidence required:** provide MDM enrollment and compliance evidence for workforce devices.
- [ ] Visitor procedures enforced — **External evidence required:** provide office or data-center visitor procedures; document the inherited responsibility when using cloud hosting.
- [ ] Security awareness training implemented — **External evidence required:** retain hire-date and annual training completion records.

## Product security

- [x] Data encryption utilized — AES-256-GCM field encryption protects MFA secrets, provider credentials, notifier credentials, OAuth secrets, and webhook secrets. See `internal/platform/crypto/encrypt.go` and `SECURITY.md`. The production database or storage layer must also enable encryption at rest.
- [ ] Control self-assessments conducted — **External evidence required:** conduct at least annual control reviews, record findings, and track corrective actions to their SLAs.
- [ ] Penetration testing performed — **External evidence required:** commission at least annual penetration testing and retain the report and remediation plan.
- [x] Data transmission encrypted — production database and Redis TLS requirements are enforced or documented in `internal/platform/config/db.go`, `internal/platform/config/redis.go`, `SECURITY.md`, and `docs/documentations/devops/operator-runbook.md`; outbound webhooks require HTTPS in `internal/webhook/security_url.go`.
- [x] Vulnerability and system monitoring procedures established — the security policy, daily CodeQL/Semgrep/Snyk/Gitleaks workflow, CI security checks, OpenTelemetry guidance, and monitoring stack establish app-level vulnerability and system monitoring. See `SECURITY.md`, `.github/workflows/security.yml`, `.github/workflows/ci.yml`, and `docs/documentations/observability/opentelemetry.md`.

## Internal security procedures

- [ ] Continuity and Disaster Recovery plans established — **Partial:** recovery and backup steps exist in `docs/documentations/devops/operator-runbook.md`; a business continuity plan with owners, communications, dependencies, RTOs, and RPOs is still required.
- [ ] Continuity and Disaster Recovery plans tested — **External evidence required:** run at least annual restore and continuity exercises and retain results and corrective actions.
- [ ] Cybersecurity insurance maintained — **External evidence required:** retain current policy and coverage evidence.
- [ ] Configuration management system established — **Partial:** configuration is centralized under `internal/platform/config/` and deployment settings are documented, but a formal configuration baseline, approval procedure, and production drift evidence are still needed.
- [ ] Change management procedures enforced — **Partial:** `.github/workflows/ci.yml` tests, lints, scans, and builds changes; branch protection, reviewer approval, deployment authorization, and emergency-change evidence must be configured and retained externally.
- [ ] Production deployment access restricted — **External evidence required:** provide deployment IAM and access-review evidence.
- [x] Development lifecycle established — code structure, testing standards, migration rules, CI, static analysis, and security scanning are documented and automated in `docs/contributing/`, `.github/workflows/ci.yml`, and `.github/workflows/security.yml`.
- [ ] SOC 2 - System Description — **Missing:** write the audit-period system description, boundaries, infrastructure, people, data, software, procedures, and subservice organizations.
- [ ] Whistleblower policy established — **External evidence required:** publish a policy and anonymous reporting channel.
- [ ] Board oversight briefings conducted — **External evidence required:** retain at least annual cybersecurity and privacy briefing records.
- [ ] Board charter documented — **External evidence required:** retain a charter assigning internal-control oversight.
- [ ] Board expertise developed — **External evidence required:** document board expertise or external advisor support.
- [ ] Board meetings conducted — **External evidence required:** retain meeting cadence, independence, and minutes.
- [x] Backup processes established — Postgres backup targets, frequency, RPO, RTO, and restoration steps are documented in `docs/documentations/devops/operator-runbook.md`.
- [ ] System changes externally communicated — **Partial:** repository releases can communicate changes, but a defined customer-notification process for critical production changes is not evidenced.
- [ ] Management roles and responsibilities defined — **External evidence required:** assign control ownership and retain a roles-and-responsibilities policy.
- [ ] Organization structure documented — **External evidence required:** maintain an organization chart and reporting lines.
- [ ] Roles and responsibilities specified — **External evidence required:** assign security-control responsibilities in job descriptions or policy.
- [ ] Security policies established and reviewed — **Partial:** `SECURITY.md` exists; an approved policy set, owners, annual review dates, and review evidence are still required.
- [x] Support system available — `SECURITY.md` provides a private security reporting channel and response targets; GitHub provides the public project support and issue channel.
- [ ] System changes communicated — **External evidence required:** define and evidence internal change communications to authorized users.
- [ ] Access reviews conducted — **External evidence required:** conduct at least quarterly reviews of production, source-control, cloud, database, and other in-scope access.
- [ ] Access requests required — **External evidence required:** retain role-based or manager-approved workforce access requests; product RBAC alone is insufficient.
- [ ] Incident response plan tested — **External evidence required:** conduct and document at least annual tabletop or technical exercises.
- [ ] Incident response policies established — **Partial:** `SECURITY.md` defines vulnerability intake and response targets, but a full security and privacy incident response plan, severity model, roles, communications, evidence handling, and breach-notification procedure are still needed.
- [ ] Incident management procedures followed — **External evidence required:** retain an incident register, timelines, communications, resolution evidence, and post-incident actions.
- [ ] Physical access processes established — **External evidence required:** document office/data-center controls and inherited cloud-provider controls.
- [ ] Data center access reviewed — **External evidence required:** retain annual review evidence or the relevant cloud-provider assurance report.
- [ ] Company commitments externally communicated — **Partial:** `SECURITY.md` publishes security response targets, but customer security commitments must be included in applicable terms, agreements, or a public trust statement.
- [x] External support resources available — repository documentation, the operator runbook, OpenAPI specification, and security reporting instructions provide technical support resources. See `README.md`, `docs/`, `docs/openapi.yaml`, and `SECURITY.md`.
- [x] Service description communicated — product purpose, features, architecture, and API behavior are documented in `README.md`, `docs/overview.md`, `docs/features.md`, and `docs/openapi.yaml`.
- [ ] Risk assessment objectives specified — **External evidence required:** define security, availability, confidentiality, processing-integrity, and privacy objectives used for risk assessment.
- [ ] Risks assessments performed — **External evidence required:** perform and approve at least annual risk and fraud assessments and track treatment actions.
- [ ] Risk management program established — **External evidence required:** document risk identification, scoring, acceptance, mitigation, ownership, and review procedures.
- [ ] Third-party agreements established — **External evidence required:** retain vendor agreements with applicable confidentiality, security, and privacy terms.
- [ ] Vendor management program established — **External evidence required:** maintain a critical-vendor inventory, due diligence, requirements, risk ratings, and annual reviews.
- [ ] Vulnerabilities scanned and remediated — **Partial:** daily code, dependency, and secret scans plus CI gosec checks are configured in `.github/workflows/security.yml` and `.github/workflows/ci.yml`; quarterly external host scanning and finding-to-remediation SLA evidence are still required.

## Data and privacy

- [x] Data retention procedures established — configurable auth-event retention and tenant purge runners are implemented in `internal/authevent/service_retention.go` and `internal/tenant/retention.go`; logging and retention rationale is documented in `docs/documentations/settings/logging-and-audit.md`.
- [x] Customer data deleted upon leaving — authorized tenant deletion cascades through tenant-owned records in `internal/tenant/service_tenant.go`, `internal/tenant/repository_tenant.go`, and `internal/tenant/unit_of_work.go`; deleted tenant records are later purged by `internal/tenant/retention.go`.
- [ ] Data classification policy established — **External evidence required:** define classification levels, handling rules, owners, labeling, access restrictions, retention, and disposal requirements.

## Priority follow-up

1. Create the foundational policy set: information security, access control,
   incident response, business continuity/disaster recovery, risk management,
   vendor management, data classification/retention, change management, and
   vulnerability management.
2. Assign an owner and review cadence to every control and policy.
3. Enable and retain production evidence for IAM, MFA, encryption, backups,
   monitoring, alerting, patching, firewall rules, endpoint management, and
   quarterly access reviews.
4. Schedule annual risk assessment, penetration test, incident-response
   exercise, and disaster-recovery restore test.
5. Track findings and remediation SLAs in one evidence system, and link the
   resulting records beside the relevant checklist items above.

## Review record

| Review date | Reviewer | Scope | Notes |
|---|---|---|---|
| 2026-06-21 | Codex repository review | Source, project documentation, and CI configuration | Initial checklist; operational and company evidence not assessed |
