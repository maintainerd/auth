package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/federation"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"github.com/maintainerd/maintainerd-auth/internal/invite"
	"github.com/maintainerd/maintainerd-auth/internal/mfa"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/geoip"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/setup"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"github.com/maintainerd/maintainerd-auth/internal/webhook"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// svcs holds every service instance. Private to the app package.
type svcs struct {
	serviceService               iam.ServiceService
	apiService                   iam.APIService
	permissionService            iam.PermissionService
	tenantService                tenant.TenantService
	tenantMemberService          tenant.TenantMemberService
	idpService                   idp.IdentityProviderService
	clientService                client.ClientService
	roleService                  iam.RoleService
	userService                  user.UserService
	registerService              authn.RegisterService
	loginService                 authn.LoginService
	profileService               user.ProfileService
	profileRepo                  user.ProfileRepository
	userSettingService           user.UserSettingService
	userConsentService           user.UserConsentService
	userTrustedDeviceService     user.UserTrustedDeviceService
	inviteService                invite.InviteService
	forgotPasswordService        authn.ForgotPasswordService
	resetPasswordService         authn.ResetPasswordService
	emailVerificationService     authn.EmailVerificationService
	magicLinkService             authn.MagicLinkService
	setupService                 setup.SetupService
	registrationFlowService      idp.RegistrationFlowService
	registrationContextService   authn.RegistrationContextService
	policyService                iam.PolicyService
	securitySettingService       secpolicy.SecuritySettingService
	ipRestrictionRuleService     secpolicy.IPRestrictionRuleService
	emailTemplateService         branding.EmailTemplateService
	smsTemplateService           branding.SMSTemplateService
	brandingService              branding.BrandingService
	tenantSettingService         tenant.TenantSettingService
	emailConfigService           notifier.EmailConfigService
	smsConfigService             notifier.SMSConfigService
	webhookEndpointService       webhook.WebhookEndpointService
	authEventService             authevent.AuthEventService
	authorizationService         iam.ServiceAuthorizationService
	oauthAuthorizeService        oauth.OAuthAuthorizeService
	oauthConnectionsService      oauth.OAuthConnectionsService
	oauthTokenService            oauth.OAuthTokenService
	oauthConsentService          oauth.OAuthConsentService
	oauthPARService              oauth.OAuthPARService
	oauthDeviceService           oauth.OAuthDeviceService
	oauthTokenExchangeService    oauth.OAuthTokenExchangeService
	oauthSessionService          oauth.OAuthSessionService
	oauthCIBAService             oauth.OAuthCIBAService
	oauthRegisterService         oauth.OAuthRegisterService
	accountService               user.AccountService
	sessionService               authn.SessionService
	smsLoginService              authn.SMSLoginService
	mfaService                   mfa.MFAService
	webAuthnService              mfa.WebAuthnService
	federationService            idp.FederationService
	eventService                 event.EventService
	eventTypeService             event.EventTypeService
	tenantEventTypeConfigService event.TenantEventTypeConfigService
	eventRouteService            event.EventRouteService
	writeGate                    *event.WriteGate
	relay                        *event.Relay
	retrier                      *event.BackgroundRetrier
	retentionRunner              *event.RetentionRunner
	webhookEndpointRepo          webhook.WebhookEndpointRepository
	webhookSubscriptionHandler   *webhook.SubscriptionHandler
	webhookReplayHandler         *webhook.ReplayHandler
	ipRestrictionRuleRepo        secpolicy.IPRestrictionRuleRepository
	auditLogger                  auditlog.ManagementAuditLogger
	keyRotationService           oauth.KeyRotationService
	tokenRevocationService       oauth.TokenRevocationService
	tokenRevocationRepo          oauth.OAuthTokenRevocationRepository
	wifService                   federation.WorkloadIdentityFederationService
	dataErasureService           user.DataErasureService
	accountLinkService           authn.AccountLinkRequestService
}

// listenerChecker adapts webhook and event-route repos for the write gate.
type listenerChecker struct {
	webhookEndpointRepo webhook.WebhookEndpointRepository
	eventRouteRepo      event.EventRouteRepository
}

func (c *listenerChecker) HasAnyActiveListener(tenantID int64) (bool, error) {
	endpoints, err := c.webhookEndpointRepo.FindActiveByTenantID(tenantID)
	if err != nil {
		return false, err
	}
	if len(endpoints) > 0 {
		return true, nil
	}

	routes, err := c.eventRouteRepo.FindEnabledByTenantID(tenantID)
	if err != nil {
		return false, err
	}
	return len(routes) > 0, nil
}

func initServices(ctx context.Context, db *gorm.DB, r *repos, appCache *cache.Cache, redisClient *redis.Client) (*svcs, error) {
	// Create event infrastructure first — write gate, relay, event service
	lc := &listenerChecker{
		webhookEndpointRepo: r.webhookEndpointRepo,
		eventRouteRepo:      r.eventRouteRepo,
	}

	writeGate := event.NewWriteGate(
		r.eventTypeRepo,
		r.tenantEventTypeRepo,
		lc,
		r.eventRouteRepo,
		redisClient,
	)

	// RabbitMQ broker publisher. Reads RABBITMQ_URL from the environment.
	// When the variable is absent the publisher is disabled (a no-op) and
	// activates the moment RABBITMQ_URL is set — matching the "disables
	// cleanly when RabbitMQ config is absent" contract.
	amqpCfg := event.NewAMQPConfigFromEnv()
	amqpPublish, _, amqpErr := event.ConnectAMQP(ctx, amqpCfg)
	if amqpErr != nil {
		return nil, fmt.Errorf("init services: amqp: %w", amqpErr)
	}
	brokerPublisher := event.NewRabbitMQPublisher(amqpPublish)
	tenantRefResolver := newTenantRefResolver(r.tenantRepo)
	eventPublicIDs := newEventPublicIDResolver(tenantRefResolver, r.userRepo)

	deliveryAdapter := event.NewDeliveryAdapter(
		func(ctx context.Context, outbox *event.Outbox) error {
			return deliverToWebhooks(ctx, outbox, r.webhookEndpointRepo, r.deliveryHistoryRepo, r.webhookEndpointEventRepo, eventPublicIDs)
		},
		newBrokerDeliverFn(brokerPublisher, r.eventTypeRepo, r.eventRouteRepo, eventPublicIDs),
	)

	// Webhook management handlers (subscription + replay) that the REST router
	// wires onto /webhook-endpoints. Built here because they need the webhook
	// repos and the shared replay delivery path.
	webhookSubscriptionHandler := webhook.NewSubscriptionHandler(r.webhookEndpointEventRepo, r.webhookEndpointRepo, r.eventTypeRepo)
	webhookReplayHandler := webhook.NewReplayHandler(
		r.deliveryHistoryRepo,
		r.webhookEndpointRepo,
		newReplayFn(r.outboxRepo, r.deliveryHistoryRepo, r.webhookEndpointRepo, eventPublicIDs),
	)

	relay := event.NewRelay(
		r.outboxRepo,
		deliveryAdapter.DeliverWebhook,
		deliveryAdapter.DeliverBroker,
	)

	// Durable retry engine: re-attempts pending delivery_history rows driven by
	// next_retry_time, surviving process restarts. Shares the single attemptOnce
	// delivery path so state transitions and quarantine stay consistent.
	retrier := event.NewBackgroundRetrier(
		&deliveryRetrierAdapter{historyRepo: r.deliveryHistoryRepo, endpointRepo: r.webhookEndpointRepo},
		newRetryDeliveryFn(r.outboxRepo, r.webhookEndpointRepo, r.deliveryHistoryRepo, eventPublicIDs),
	)

	// Retention: purge published outbox rows (>7d) and delivery history (>90d).
	retentionRunner := event.NewRetentionRunner(r.outboxRepo, r.deliveryHistoryRepo)
	retentionRunner.Start()

	eventSvc := event.NewEventService(r.outboxRepo, writeGate, relay)

	eventTypeSvc := event.NewEventTypeServiceImpl(r.eventTypeRepo)
	tenantEventTypeConfigSvc := event.NewTenantEventTypeConfigService(db, r.tenantEventTypeRepo, r.eventTypeRepo, writeGate)
	eventRouteSvc := event.NewEventRouteService(db, r.eventRouteRepo, r.eventTypeRepo, writeGate)

	tenantSettingSvc := tenant.NewTenantSettingService(r.tenantSettingRepo)

	// Create authEventService — it is injected into other services that
	// need structured audit logging.
	// Auth events are persisted durably and are NOT fanned out to webhooks (they
	// are not integration events). Integration events are delivered via the
	// outbox → relay → {webhook, broker} plane wired above.
	authEventSvc := authevent.NewAuthEventService(r.authEventRepo, tenantSettingSvc)

	// Revoking a session must also revoke the OAuth refresh tokens minted from
	// it, or the session ends while a long-lived credential keeps issuing fresh
	// access tokens. The scope of each revoke is chosen by the caller — see
	// refreshRevokerAdapter.
	refreshRevoker := refreshRevokerAdapter{tokens: r.oauthRefreshTokenRepo}
	sessionSvc := authn.NewSessionService(r.userSessionRepo, refreshRevoker)
	iamClientRepo := newIAMClientRepo(db)
	iamTenantRepo := newIAMTenantRepo(db)
	iamUserRepo := newIAMUserRepo(db)
	clientAPIRepo := newClientAPIRepo(db)
	clientPermissionRepo := newClientPermissionRepo(db)
	clientIDPRepo := newClientIDPRepo(db)
	clientTenantRepo := newClientTenantRepo(db)
	clientUserRepo := newClientUserRepo(db)
	clientRoleRepoAdapter := newClientRoleRepo(db)
	userTenantRepo := newUserTenantRepo(db)
	userRoleRepo := newUserRoleRepo(db)
	userClientRepo := newUserClientRepo(db)
	userIDPRepo := newUserIDPRepo(db)
	userBackupCodeRepo := newUserMFABackupCodeRepo(db)
	idpTenantRepo := newIDPTenantRepo(db)
	idpUserRepo := newIDPUserRepo(db)
	idpUserIdentityRepo := newIDPUserIdentityRepo(db)
	idpClientRepo := newIDPClientRepo(db)
	idpUserRoleRepo := newIDPUserRoleRepo(db)
	idpRoleRepo := newIDPRoleRepo(db)
	mfaUserRepo := newMFAUserRepo(db)
	oauthClientRepo := newOAuthClientRepo(db)
	oauthClientURIRepo := newOAuthClientURIRepo(db)
	oauthTenantRepo := newOAuthTenantRepo(db)
	oauthUserRepo := newOAuthUserRepo(db)
	oauthUserIdentityRepo := newOAuthUserIdentityRepo(db)
	authzInvalidator := iam.NewDBAuthorizationTokenInvalidator(db, appCache)
	tenantUOW := tenant.NewGormUnitOfWork(db, r.tenantRepo, r.tenantMemberRepo, tenantCascadeModels())
	middleware.SetSessionValidator(sessionSvc)

	// Let CORS honour the `cors_origin_uri` entries operators register against
	// their OAuth clients. Without this a third-party SPA on its own domain
	// cannot POST to /oauth/token, so authorization-code + PKCE dies at the
	// exchange even though the code is valid — and registering the origin in the
	// admin console has no effect.
	middleware.SetCORSOriginResolver(client.NewCORSOriginResolver(db))

	// Tokens carry the tenant's opaque UUID in the `tenant_id` claim (never the
	// internal PK); this resolver lets the mint layer stamp the UUID and the JWT
	// parse layer resolve it back to the internal id every scoping check expects.
	shared.SetTenantRefResolver(tenantRefResolver)

	webAuthnSvc, err := mfa.NewWebAuthnService(db, mfaUserRepo, r.mfaWebAuthnCredRepo, appCache, authEventSvc, r.webauthnChallengeRepo)
	if err != nil {
		return nil, err
	}

	// Built before the login service so login can verify the MFA second step
	// (acr=2 elevation) via the MFAFactorAuthenticator interface.
	mfaSvc := mfa.NewMFAService(db, mfaUserRepo, r.totpSecretRepo, r.mfaWebAuthnCredRepo, webAuthnSvc, r.userBackupCodeRepo, r.mfaPhoneRepo, r.emailOTPRepo, r.smsOtpRepo, r.securitySettingRepo, authEventSvc)
	// Trusted-device geolocation uses a local MaxMind GeoLite2 DB (GEOIP_DB_PATH),
	// so no client IP leaves the server. Disabled (no-op) when unset/unreadable.
	if geoResolver, geoErr := geoip.New(config.GetEnvOrDefault("GEOIP_DB_PATH", "")); geoErr != nil {
		slog.Warn("geoip: trusted-device geolocation disabled", "error", geoErr)
	} else {
		mfaSvc.SetGeoResolver(geoResolver)
	}
	emailVerificationSvc := authn.NewEmailVerificationService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo, newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), appCache, r.securitySettingRepo)
	// The user service's session-admin operations (list/revoke) must read the
	// canonical user_sessions store (owned by authn), not the legacy user_tokens
	// session rows the login flow no longer writes. Wrap the token repo so its
	// session methods are backed by user_sessions while token methods are unchanged.
	userSessionRepo := authn.NewUserSessionRepository(db)
	userSessionTokenRepo := newUserSessionBackedTokenRepo(r.userTokenRepo, userSessionRepo)
	userSvc := user.NewUserService(db, r.userRepo, r.userIdentityRepo, r.userRoleRepo, userRoleRepo, userTenantRepo, userIDPRepo, userClientRepo, appCache, userSessionTokenRepo, r.securitySettingRepo, r.userPasswordHistoryRepo, authEventSvc, eventSvc)

	// idpEmailDomainRepo backs the identity_provider_email_domains child table
	// (home-realm discovery + create/update domain membership).
	idpEmailDomainRepo := idp.NewIdentityProviderEmailDomainRepository(db)

	federationSvc := idp.NewFederationService(db, idpUserRepo, idpUserIdentityRepo, r.idpRepo, idpEmailDomainRepo, idpClientRepo, idpUserRoleRepo, idpRoleRepo, authEventSvc, eventSvc, r.securitySettingRepo, appCache, sessionSvc)
	// Account linking parks its in-flight state in the OAuth broker-session
	// table, discriminated by purpose. idp cannot import oauth, so the adapter
	// is supplied here.
	federationSvc.SetIdentityLinkStore(identityLinkStoreAdapter{sessions: oauth.NewOAuthBrokerSessionRepository(db)})

	s := &svcs{
		serviceService:      iam.NewServiceService(db, r.serviceRepo, r.apiRepo, r.servicePolicyRepo, r.policyRepo, authEventSvc),
		apiService:          iam.NewAPIService(db, r.apiRepo, r.serviceRepo, eventSvc),
		permissionService:   iam.NewPermissionService(db, r.permissionRepo, r.apiRepo, r.roleRepo, iamClientRepo, appCache, eventSvc, authzInvalidator),
		tenantService:       tenant.NewTenantService(r.tenantRepo, tenantUOW, eventSvc, tenantSeederAdapter{}),
		tenantMemberService: tenant.NewTenantMemberService(r.tenantMemberRepo, newTenantUserReader(r.userRepo), r.tenantRepo, tenantUOW, eventSvc, newTenantUserProvisioner(userSvc)),
		idpService:          idp.NewIdentityProviderService(db, r.idpRepo, idpEmailDomainRepo, r.idpAllowedAudienceRepo, idpTenantRepo, idpUserRepo, appCache),
		clientService: client.NewClientService(db, r.clientRepo, r.clientURIRepo, clientIDPRepo, clientPermissionRepo, r.clientPermissionRepo, r.clientAPIRepo, r.clientRoleRepo, clientRoleRepoAdapter, clientAPIRepo, clientUserRepo, clientTenantRepo, authEventSvc, eventSvc,
			// Connection changes are authorization changes: reachability is
			// resolved through client_identity_providers, so a disabled or
			// removed connection must drop cached user contexts immediately
			// rather than staying live for the cache TTL.
			appCache),
		roleService: iam.NewRoleService(db, r.roleRepo, r.permissionRepo, r.rolePermissionRepo, iamUserRepo, iamTenantRepo, appCache, authEventSvc, eventSvc, authzInvalidator),
		userService: userSvc,
		registerService: authn.NewRegistrationService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserRoleRepoAdapter(r.userRoleRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnRoleRepoAdapter(r.roleRepo), newAuthnInviteRepoAdapter(r.inviteRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo), newAuthnRegistrationFlowRoleRepoAdapter(db, r.registrationFlowRoleRepo, r.registrationFlowRepo),
			authn.WithEmailVerificationService(emailVerificationSvc),
			authn.WithConsentRecorder(user.NewUserConsentService(r.userConsentRepo)),
			// Registration signs the user in, so it gets the same session
			// handling as login — without this a registered user has no
			// user_sessions row and sits outside the whole session layer.
			authn.WithRegisterSessionService(sessionSvc),
		),
		loginService:             authn.NewLoginService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc, sessionSvc, r.securitySettingRepo, appCache),
		sessionService:           sessionSvc,
		profileService:           user.NewProfileService(db, r.profileRepo, r.userRepo, user.WithProfilePictureRepository(r.profilePictureRepo)),
		profileRepo:              r.profileRepo,
		userSettingService:       user.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		userConsentService:       user.NewUserConsentService(r.userConsentRepo),
		userTrustedDeviceService: user.NewUserTrustedDeviceService(r.userTrustedDeviceRepo),
		inviteService:            invite.NewInviteService(db, r.inviteRepo, newInviteClientRepo(db), r.emailTemplateRepo, newInviteRegistrationFlowRepo(db)),
		forgotPasswordService:    authn.NewForgotPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo),
		resetPasswordService:     authn.NewResetPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo), userSessionRepo, refreshRevoker),
		emailVerificationService: emailVerificationSvc,
		magicLinkService:         authn.NewMagicLinkService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.emailTemplateRepo),
		setupService: setup.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.clientRepo, r.roleRepo, r.userRoleRepo, r.userIdentityRepo, r.profileRepo,
			setup.ControlRegistrationDeps{
				ServiceRepo:        r.serviceRepo,
				PolicyRepo:         r.policyRepo,
				ServicePolicyRepo:  r.servicePolicyRepo,
				APIRepo:            r.apiRepo,
				PermissionRepo:     r.permissionRepo,
				RolePermissionRepo: r.rolePermissionRepo,
				ClientURIRepo:      r.clientURIRepo,
			},
		),
		registrationContextService: authn.NewRegistrationContextService(
			newAuthnClientRepoAdapter(r.clientRepo),
			newAuthnRegistrationFlowRoleRepoAdapter(db, r.registrationFlowRoleRepo, r.registrationFlowRepo),
			r.securitySettingRepo,
		),
		registrationFlowService:      idp.NewRegistrationFlowService(db, r.registrationFlowRepo, r.registrationFlowRoleRepo, idpRoleRepo, idpClientRepo, idpUserRepo, idpUserRoleRepo, newIDPRegistrationFlowInviteCounter(db), newIDPRolePermissionNameReader(db)),
		policyService:                iam.NewPolicyService(db, r.policyRepo, r.serviceRepo, r.apiRepo, eventSvc, authEventSvc),
		securitySettingService:       secpolicy.NewSecuritySettingService(db, r.securitySettingRepo, r.securitySettingsAuditRepo),
		ipRestrictionRuleService:     secpolicy.NewIPRestrictionRuleService(db, r.ipRestrictionRuleRepo),
		emailTemplateService:         branding.NewEmailTemplateService(r.emailTemplateRepo),
		smsTemplateService:           branding.NewSMSTemplateService(r.smsTemplateRepo),
		brandingService:              branding.NewBrandingService(r.brandingRepo),
		tenantSettingService:         tenantSettingSvc,
		emailConfigService:           notifier.NewEmailConfigService(r.emailConfigRepo),
		smsConfigService:             notifier.NewSMSConfigService(r.smsConfigRepo),
		webhookEndpointService:       webhook.NewWebhookEndpointService(r.webhookEndpointRepo),
		authEventService:             authEventSvc,
		authorizationService:         iam.NewServiceAuthorizationService(r.serviceRepo, r.servicePolicyRepo),
		oauthAuthorizeService:        oauth.NewOAuthAuthorizeService(db, oauthClientRepo, oauthClientURIRepo, r.oauthAuthCodeRepo, r.oauthConsentGrantRepo, r.oauthConsentChallengeRepo, authEventSvc, oauth.NewOAuthAuthorizeRequestRepository(db), r.securitySettingRepo),
		oauthConnectionsService:      oauth.NewOAuthConnectionsService(db, oauthClientRepo, r.securitySettingRepo, branding.NewClientBrandingResolver(db)),
		oauthTokenService:            oauth.NewOAuthTokenService(db, oauthClientRepo, r.oauthAuthCodeRepo, r.oauthRefreshTokenRepo, oauthUserRepo, oauthUserIdentityRepo, authEventSvc, appCache, r.securitySettingRepo),
		oauthConsentService:          oauth.NewOAuthConsentService(r.oauthConsentGrantRepo, authEventSvc),
		oauthPARService:              oauth.NewOAuthPARService(db, oauthClientRepo, oauthClientURIRepo, r.oauthPARRequestRepo, authEventSvc, r.securitySettingRepo),
		oauthDeviceService:           oauth.NewOAuthDeviceService(db, oauthClientRepo, r.oauthDeviceCodeRepo, oauthUserRepo, oauthUserIdentityRepo, authEventSvc, r.securitySettingRepo),
		oauthTokenExchangeService:    oauth.NewOAuthTokenExchangeService(db, oauthClientRepo, oauthUserRepo, authEventSvc, r.tokenExchangeRepo, r.securitySettingRepo),
		oauthSessionService:          oauth.NewOAuthSessionService(db, oauthClientRepo, oauthUserRepo, r.oauthRefreshTokenRepo, authEventSvc),
		oauthCIBAService:             oauth.NewOAuthCIBAService(db, oauthClientRepo, r.oauthCIBARequestRepo, oauthUserRepo, authEventSvc, r.securitySettingRepo),
		oauthRegisterService:         oauth.NewOAuthRegisterService(db, oauthClientRepo, oauthClientURIRepo, oauthTenantRepo, authEventSvc),
		accountService:               user.NewAccountService(db, r.userRepo, r.userTokenRepo, r.profileRepo, r.userSettingRepo, userRoleRepo, userClientRepo, userBackupCodeRepo, r.userIdentityRepo, userIDPRepo, authEventSvc, r.securitySettingRepo, r.smsOtpRepo, r.userPasswordHistoryRepo, userSvc, sessionRevokerAdapter{sessions: userSessionRepo}, refreshRevoker),
		smsLoginService:              authn.NewSMSLoginService(db, newAuthnUserRepoAdapter(r.userRepo), r.smsOtpRepo, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc, sessionSvc, r.securitySettingRepo),
		mfaService:                   mfaSvc,
		webAuthnService:              webAuthnSvc,
		federationService:            federationSvc,
		eventService:                 eventSvc,
		eventTypeService:             eventTypeSvc,
		tenantEventTypeConfigService: tenantEventTypeConfigSvc,
		eventRouteService:            eventRouteSvc,
		writeGate:                    writeGate,
		relay:                        relay,
		retrier:                      retrier,
		retentionRunner:              retentionRunner,
		webhookEndpointRepo:          r.webhookEndpointRepo,
		webhookSubscriptionHandler:   webhookSubscriptionHandler,
		webhookReplayHandler:         webhookReplayHandler,
		ipRestrictionRuleRepo:        r.ipRestrictionRuleRepo,
		auditLogger:                  auditlog.NewManagementAuditLogger(r.auditLogRepo),
		// db is what makes Rotate work: without it the service is JWKS-serving only
		// and the operator-triggered rotation endpoint returns "no database handle".
		keyRotationService:     oauth.NewKeyRotationService(r.signingKeyRepo, db),
		tokenRevocationService: oauth.NewTokenRevocationService(r.tokenRevocationRepo),
		tokenRevocationRepo:    r.tokenRevocationRepo,
		wifService:             federation.NewWorkloadIdentityFederationService(db, r.wifRepo, newFederationExchangeAuditor(r.tokenExchangeRepo)),
		dataErasureService:     user.NewDataErasureService(r.dataErasureRequestRepo, userSvc),
		accountLinkService:     authn.NewAccountLinkRequestService(r.accountLinkRequestRepo, newAuthnUserRepoAdapter(r.userRepo), newAccountLinkIdentityLinker(r.userIdentityRepo)),
	}
	// Wire the broker provider resolver so the oauth broker flow (idp_hint →
	// upstream provider) can resolve provider authorize endpoints + client_ids
	// without importing the idp package directly.
	oauth.SetBrokerProviderResolver(&oauthBrokerProviderResolver{federation: s.federationService})

	// Wire the broker callback resolver so the oauth broker flow can exchange an
	// upstream provider code and resolve/provision the user.
	oauth.SetBrokerCallbackResolver(&oauthBrokerCallbackResolver{federation: s.federationService})

	// Inject the account-link service so the federation provisioning path can
	// create a confirmation request on email collision instead of silently merging.
	s.federationService.SetAccountLinkService(s.accountLinkService)

	// Wire the account-link verifier so BrokerResume can validate confirmed tokens.
	oauth.SetBrokerAccountLinkVerifier(&accountLinkVerifierAdapter{repo: r.accountLinkRequestRepo})

	// Sweep expired broker sessions every 5 minutes so single-use short-lived
	// rows don't accumulate in the database.
	oauth.SweepExpiredBrokerSessions(context.Background(), db, 5*time.Minute)

	// Inject event service into ServiceService (uses setter to avoid breaking test constructors)
	iam.SetServiceEventService(s.serviceService, eventSvc)
	// Inject the policy version history repo (snapshots before-state on every
	// Update + powers the read endpoints in PolicyHistoryHandler).
	iam.SetPolicyVersionHistory(s.policyService, r.policyVersionHistoryRepo)
	// Inject the tenant directory so policy writes can enforce the MRN tenant
	// boundary: a regular tenant may only carry MRN resources scoped to its own
	// tenant segment; "*", platform scope, and other tenants' literals are
	// reserved for the system tenant (the control plane). Without this wiring
	// MRN-bearing policies are refused outright (fail closed).
	iam.SetPolicyTenantDirectory(s.policyService, s.tenantService)
	// Inject the MFA factor verifier so login can run the MFA second step (acr=2).
	s.loginService.SetMFAFactorAuthenticator(mfaSvc)
	s.loginService.SetUserLockoutRepository(r.userLockoutRepo)
	s.loginService.SetTokenRevoker(s.tokenRevocationService)
	// Wire client permission resolver for M2M token issuance.
	s.oauthTokenService.SetClientPermissionResolver(newClientPermissionResolver(db))
	// Wire the workload identity federation exchanger so the /oauth/token
	// endpoint can exchange external OIDC workload tokens (section 3.21).
	oauth.SetWorkloadIdentityExchanger(newOAuthWorkloadIdentityExchanger(s.wifService))
	// Magic-link possession is the first factor; delegate MFA policy decisions
	// and policy-aware session issuance to the normal login service.
	s.magicLinkService.SetLoginCoordinator(s.loginService)
	// SMS OTP is likewise a single first factor — route it through the same login
	// service so mode=enforced is honored on SMS login instead of being bypassed.
	s.smsLoginService.SetMFACoordinator(s.loginService)
	// Wire the account-lockout notification hook so the event system is notified.
	security.OnAccountLockout = func(ctx context.Context, identifier string) {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			EventType:   authevent.AuthEventTypeLoginLock,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Account locked due to too many failed login attempts: " + identifier),
		})
	}
	// Route threat signals into the auth-event system so new_device_notification
	// and impossible_travel actually notify (fan out to webhooks) instead of only
	// writing a local log line. Both are gated by their tenant config toggles in
	// the threat engine before these fire.
	security.OnNewDeviceLogin = func(ctx context.Context, tenantID, userID int64, ip, userAgent string) {
		uid := userID
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    tenantID,
			ActorUserID: &uid,
			IPAddress:   ip,
			UserAgent:   ptr.PtrOrNil(userAgent),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeNewDevice,
			Severity:    authevent.AuthEventSeverityInfo,
			Result:      authevent.AuthEventResultSuccess,
			Description: ptr.Ptr("Sign-in from a new device"),
		})
	}
	security.OnImpossibleTravel = func(ctx context.Context, tenantID, userID int64, ip, userAgent string) {
		uid := userID
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    tenantID,
			ActorUserID: &uid,
			IPAddress:   ip,
			UserAgent:   ptr.PtrOrNil(userAgent),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeImpossibleTravel,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultSuccess,
			Description: ptr.Ptr("Rapid sign-in from a different location"),
		})
	}
	// Durable AUTHZ/failure record for an authenticated-but-unauthorized request
	// (NIST AU-2 / PCI 10.2.4). Fired by PermissionMiddleware on a 403; the denial
	// is also always metered (security_denials_total) for alerting.
	middleware.OnAccessDenied = func(ctx context.Context, tenantID, actorUserID int64, ip string, requiredPermissions []string) {
		uid := actorUserID
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    tenantID,
			ActorUserID: &uid,
			IPAddress:   ip,
			Category:    authevent.AuthEventCategoryAuthz,
			EventType:   authevent.AuthEventTypeAuthzFail,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr(fmt.Sprintf("Access denied: missing one of required permissions %v", requiredPermissions)),
		})
	}
	// Backup-code recovery completes a login, so its token has to be bound to a
	// real session — otherwise it cannot be revoked and the session middleware
	// rejects it on the next request.
	s.accountService.SetSessionCreator(sessionCreatorAdapter{sessions: s.sessionService})

	// Without this the OAuth token service falls back to a hardcoded
	// amr=["pwd"] / acr=1, so a user who just completed TOTP or a passkey gets a
	// token asserting a password login and is re-challenged by every step-up
	// route immediately after authenticating.
	s.oauthTokenService.SetSessionAuthContextResolver(oauth.NewUserSessionAuthContextResolver(db))

	// OIDC Back-Channel Logout §2.6 requires a logout token's jti to be
	// single-use. The in-process guard cannot hold that across replicas, so back
	// it with the same shared store used for access-token revocation.
	oauth.SetLogoutTokenReplayStore(appCache)

	// The `iss` claim is the client's domain, so the allowlist the JWT validator
	// enforces can only be built from the registered clients.
	seedAcceptedIssuers(db)

	return s, nil
}

func tenantCascadeModels() []any {
	return []any{
		&oauth.OAuthCIBARequest{},
		&oauth.OAuthDeviceCode{},
		&oauth.OAuthPARRequest{},
		&oauth.OAuthConsentChallenge{},
		&oauth.OAuthConsentGrant{},
		&oauth.OAuthRefreshToken{},
		&oauth.OAuthAuthorizationCode{},

		&webhook.WebhookEndpoint{},
		&webhook.DeliveryHistory{},
		&notifier.SMSConfig{},
		&notifier.EmailConfig{},
		&branding.SMSTemplate{},
		&branding.EmailTemplate{},
		&branding.Branding{},

		&secpolicy.SecuritySettingsAudit{},
		&secpolicy.IPRestrictionRule{},
		&secpolicy.SecuritySetting{},

		&invite.Invite{},
		&idp.RegistrationFlow{},
		&idp.IdentityProvider{},

		&client.ClientURI{},
		&client.Client{},

		&iam.Permission{},
		&iam.API{},
		&iam.Policy{},
		&iam.Role{},
		&iam.Service{},

		&user.UserIdentity{},

		&tenant.TenantSetting{},
		&tenant.TenantMember{},
	}
}
