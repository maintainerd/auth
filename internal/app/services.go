package app

import (
	"context"
	"fmt"
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
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/setup"
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

func initServices(db *gorm.DB, r *repos, appCache *cache.Cache, redisClient *redis.Client) (*svcs, error) {
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
	amqpPublish, _, amqpErr := event.ConnectAMQP(amqpCfg)
	if amqpErr != nil {
		return nil, fmt.Errorf("init services: amqp: %w", amqpErr)
	}
	brokerPublisher := event.NewRabbitMQPublisher(amqpPublish)

	deliveryAdapter := event.NewDeliveryAdapter(
		func(ctx context.Context, outbox *event.Outbox) error {
			return deliverToWebhooks(ctx, outbox, r.webhookEndpointRepo, r.deliveryHistoryRepo, r.webhookEndpointEventRepo)
		},
		newBrokerDeliverFn(brokerPublisher, r.eventTypeRepo, r.eventRouteRepo),
	)

	// Webhook management handlers (subscription + replay) that the REST router
	// wires onto /webhook-endpoints. Built here because they need the webhook
	// repos and the shared replay delivery path.
	webhookSubscriptionHandler := webhook.NewSubscriptionHandler(r.webhookEndpointEventRepo, r.webhookEndpointRepo, r.eventTypeRepo)
	webhookReplayHandler := webhook.NewReplayHandler(
		r.deliveryHistoryRepo,
		r.webhookEndpointRepo,
		newReplayFn(r.outboxRepo, r.deliveryHistoryRepo, r.webhookEndpointRepo),
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
		newRetryDeliveryFn(r.outboxRepo, r.webhookEndpointRepo, r.deliveryHistoryRepo),
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
	// The old webhook dispatcher is deprecated; integration events are now delivered via outbox.
	authEventSvc := authevent.NewAuthEventService(r.authEventRepo, webhook.NewDispatcher(r.webhookEndpointRepo), tenantSettingSvc)

	sessionSvc := authn.NewSessionService(r.userSessionRepo)
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

	webAuthnSvc, err := mfa.NewWebAuthnService(db, mfaUserRepo, r.mfaWebAuthnCredRepo, appCache, authEventSvc, r.webauthnChallengeRepo)
	if err != nil {
		return nil, err
	}

	// Built before the login service so login can verify the MFA second step
	// (acr=2 elevation) via the MFAFactorAuthenticator interface.
	mfaSvc := mfa.NewMFAService(db, mfaUserRepo, r.totpSecretRepo, r.mfaWebAuthnCredRepo, webAuthnSvc, r.userBackupCodeRepo, r.mfaPhoneRepo, r.emailOTPRepo, r.smsOtpRepo, r.securitySettingRepo, authEventSvc)
	emailVerificationSvc := authn.NewEmailVerificationService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo, newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), appCache, r.securitySettingRepo)
	userSvc := user.NewUserService(db, r.userRepo, r.userIdentityRepo, r.userRoleRepo, userRoleRepo, userTenantRepo, userIDPRepo, userClientRepo, appCache, r.userTokenRepo, r.securitySettingRepo, r.userPasswordHistoryRepo, authEventSvc, eventSvc)

	// idpEmailDomainRepo backs the identity_provider_email_domains child table
	// (home-realm discovery + create/update domain membership).
	idpEmailDomainRepo := idp.NewIdentityProviderEmailDomainRepository(db)

	s := &svcs{
		serviceService:      iam.NewServiceService(db, r.serviceRepo, r.apiRepo, r.servicePolicyRepo, r.policyRepo, authEventSvc),
		apiService:          iam.NewAPIService(db, r.apiRepo, r.serviceRepo, eventSvc),
		permissionService:   iam.NewPermissionService(db, r.permissionRepo, r.apiRepo, r.roleRepo, iamClientRepo, appCache, eventSvc, authzInvalidator),
		tenantService:       tenant.NewTenantService(r.tenantRepo, tenantUOW, eventSvc, tenantSeederAdapter{}),
		tenantMemberService: tenant.NewTenantMemberService(r.tenantMemberRepo, newTenantUserReader(r.userRepo), r.tenantRepo, tenantUOW, eventSvc, newTenantUserProvisioner(userSvc)),
		idpService:          idp.NewIdentityProviderService(db, r.idpRepo, idpEmailDomainRepo, r.idpAllowedAudienceRepo, idpTenantRepo, idpUserRepo),
		clientService:       client.NewClientService(db, r.clientRepo, r.clientURIRepo, clientIDPRepo, clientPermissionRepo, r.clientPermissionRepo, r.clientAPIRepo, r.clientRoleRepo, clientRoleRepoAdapter, clientAPIRepo, clientUserRepo, clientTenantRepo, authEventSvc, eventSvc),
		roleService:         iam.NewRoleService(db, r.roleRepo, r.permissionRepo, r.rolePermissionRepo, iamUserRepo, iamTenantRepo, appCache, authEventSvc, eventSvc, authzInvalidator),
		userService:         userSvc,
		registerService: authn.NewRegistrationService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserRoleRepoAdapter(r.userRoleRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnRoleRepoAdapter(r.roleRepo), newAuthnInviteRepoAdapter(r.inviteRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo), newAuthnRegistrationFlowRoleRepoAdapter(r.registrationFlowRoleRepo, r.registrationFlowRepo),
			authn.WithEmailVerificationService(emailVerificationSvc),
			authn.WithConsentRecorder(user.NewUserConsentService(r.userConsentRepo)),
		),
		loginService:             authn.NewLoginService(db, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc, sessionSvc, r.securitySettingRepo, appCache),
		sessionService:           sessionSvc,
		profileService:           user.NewProfileService(db, r.profileRepo, r.userRepo),
		profileRepo:              r.profileRepo,
		userSettingService:       user.NewUserSettingService(db, r.userSettingRepo, r.userRepo),
		userConsentService:       user.NewUserConsentService(r.userConsentRepo),
		userTrustedDeviceService: user.NewUserTrustedDeviceService(r.userTrustedDeviceRepo),
		inviteService:            invite.NewInviteService(db, r.inviteRepo, newInviteClientRepo(db), r.emailTemplateRepo, newInviteRegistrationFlowRepo(db)),
		forgotPasswordService:    authn.NewForgotPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.emailTemplateRepo),
		resetPasswordService:     authn.NewResetPasswordService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), r.securitySettingRepo, newAuthnPasswordHistoryRepoAdapter(r.userPasswordHistoryRepo)),
		emailVerificationService: emailVerificationSvc,
		magicLinkService:         authn.NewMagicLinkService(db, newAuthnUserRepoAdapter(r.userRepo), newAuthnUserTokenRepoAdapter(r.userTokenRepo), newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), r.emailTemplateRepo),
		setupService: setup.NewSetupService(db, r.userRepo, r.tenantRepo, r.tenantMemberRepo, r.clientRepo, r.roleRepo, r.userRoleRepo, r.userIdentityRepo, r.profileRepo,
			setup.ControlRegistrationDeps{
				ServiceRepo:       r.serviceRepo,
				PolicyRepo:        r.policyRepo,
				ServicePolicyRepo: r.servicePolicyRepo,
			},
		),
		registrationFlowService:      idp.NewRegistrationFlowService(db, r.registrationFlowRepo, r.registrationFlowRoleRepo, idpRoleRepo, idpClientRepo),
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
		oauthTokenExchangeService:    oauth.NewOAuthTokenExchangeService(db, oauthClientRepo, oauthUserRepo, authEventSvc, oauth.NewOAuthTokenExchangeRepository(db), r.securitySettingRepo),
		oauthSessionService:          oauth.NewOAuthSessionService(db, oauthClientRepo, oauthUserRepo, r.oauthRefreshTokenRepo, authEventSvc),
		oauthCIBAService:             oauth.NewOAuthCIBAService(db, oauthClientRepo, r.oauthCIBARequestRepo, oauthUserRepo, authEventSvc, r.securitySettingRepo),
		oauthRegisterService:         oauth.NewOAuthRegisterService(db, oauthClientRepo, oauthClientURIRepo, oauthTenantRepo, authEventSvc),
		accountService:               user.NewAccountService(db, r.userRepo, r.userTokenRepo, r.profileRepo, r.userSettingRepo, userRoleRepo, userClientRepo, userBackupCodeRepo, r.userIdentityRepo, userIDPRepo, authEventSvc, r.securitySettingRepo, r.smsOtpRepo),
		smsLoginService:              authn.NewSMSLoginService(db, newAuthnUserRepoAdapter(r.userRepo), r.smsOtpRepo, newAuthnClientRepoAdapter(r.clientRepo), newAuthnUserIdentityRepoAdapter(r.userIdentityRepo), newAuthnIDPRepoAdapter(r.idpRepo), authEventSvc, sessionSvc, r.securitySettingRepo),
		mfaService:                   mfaSvc,
		webAuthnService:              webAuthnSvc,
		federationService:            idp.NewFederationService(db, idpUserRepo, idpUserIdentityRepo, r.idpRepo, idpEmailDomainRepo, idpClientRepo, idpUserRoleRepo, idpRoleRepo, authEventSvc, eventSvc, r.securitySettingRepo, appCache, sessionSvc),
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
		keyRotationService:           oauth.NewKeyRotationService(r.signingKeyRepo),
		tokenRevocationService:       oauth.NewTokenRevocationService(r.tokenRevocationRepo),
		tokenRevocationRepo:          r.tokenRevocationRepo,
		wifService:                   federation.NewWorkloadIdentityFederationService(db, r.wifRepo, newFederationExchangeAuditor(r.tokenExchangeRepo)),
		dataErasureService:           user.NewDataErasureService(r.dataErasureRequestRepo, userSvc),
		accountLinkService:           authn.NewAccountLinkRequestService(r.accountLinkRequestRepo, newAuthnUserRepoAdapter(r.userRepo), newAccountLinkIdentityLinker(r.userIdentityRepo)),
	}
	// Wire the broker provider resolver so the oauth broker flow (idp_hint →
	// upstream provider) can resolve provider authorize endpoints + client_ids
	// without importing the idp package directly.
	oauth.SetBrokerProviderResolver(&oauthBrokerProviderResolver{federation: s.federationService})

	// Wire the broker callback resolver so the oauth broker flow can exchange an
	// upstream provider code and resolve/provision the user.
	oauth.SetBrokerCallbackResolver(&oauthBrokerCallbackResolver{federation: s.federationService})

	// Sweep expired broker sessions every 5 minutes so single-use short-lived
	// rows don't accumulate in the database.
	oauth.SweepExpiredBrokerSessions(context.Background(), db, 5*time.Minute)

	// Inject event service into ServiceService (uses setter to avoid breaking test constructors)
	iam.SetServiceEventService(s.serviceService, eventSvc)
	// Inject the policy version history repo (snapshots before-state on every
	// Update + powers the read endpoints in PolicyHistoryHandler).
	iam.SetPolicyVersionHistory(s.policyService, r.policyVersionHistoryRepo)
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
	// Wire the account-lockout notification hook so the event system is notified.
	security.OnAccountLockout = func(ctx context.Context, identifier string) {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			EventType:   authevent.AuthEventTypeLoginLock,
			Severity:    authevent.AuthEventSeverityWarn,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Account locked due to too many failed login attempts: " + identifier),
		})
	}
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
