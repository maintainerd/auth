package branding

import (
	"context"

	"github.com/google/uuid"
)

type testBrandingTenantResolver struct {
	getByUUIDFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testBrandingTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(ctx, tenantUUID)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tenantUUID}, nil
}

type testBrandingService struct {
	getFn    func(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error)
	updateFn func(ctx context.Context, tenantID int64, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error)
}

func (m *testBrandingService) Get(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) {
	return m.getFn(ctx, tenantID)
}
func (m *testBrandingService) Update(ctx context.Context, tenantID int64, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL string) (*BrandingServiceDataResult, error) {
	return m.updateFn(ctx, tenantID, companyName, logoURL, faviconURL, primaryColor, secondaryColor, accentColor, fontFamily, customCSS, supportURL, privacyPolicyURL, termsOfServiceURL)
}

type testEmailTemplateService struct {
	getAllFn       func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error)
	getByUUIDFn    func(ctx context.Context, etUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error)
	createFn       func(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error)
	updateFn       func(ctx context.Context, etUUID uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error)
	updateStatusFn func(ctx context.Context, etUUID uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error)
	deleteFn       func(ctx context.Context, etUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error)
}

func (m *testEmailTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, name, status, isDefault, isSystem, page, limit, sortBy, sortOrder)
}
func (m *testEmailTemplateService) GetByUUID(ctx context.Context, etUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
	return m.getByUUIDFn(ctx, etUUID, tenantID)
}
func (m *testEmailTemplateService) Create(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) {
	return m.createFn(ctx, tenantID, name, subject, bodyHTML, bodyPlain, status, isDefault)
}
func (m *testEmailTemplateService) Update(ctx context.Context, etUUID uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error) {
	return m.updateFn(ctx, etUUID, tenantID, name, subject, bodyHTML, bodyPlain, status)
}
func (m *testEmailTemplateService) UpdateStatus(ctx context.Context, etUUID uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error) {
	return m.updateStatusFn(ctx, etUUID, tenantID, status)
}
func (m *testEmailTemplateService) Delete(ctx context.Context, etUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
	return m.deleteFn(ctx, etUUID, tenantID)
}

type testSMSTemplateService struct {
	getAllFn       func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error)
	getByUUIDFn    func(ctx context.Context, stUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error)
	createFn       func(ctx context.Context, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	updateFn       func(ctx context.Context, stUUID uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	updateStatusFn func(ctx context.Context, stUUID uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error)
	deleteFn       func(ctx context.Context, stUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error)
}

func (m *testSMSTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, name, status, isDefault, isSystem, page, limit, sortBy, sortOrder)
}
func (m *testSMSTemplateService) GetByUUID(ctx context.Context, stUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	return m.getByUUIDFn(ctx, stUUID, tenantID)
}
func (m *testSMSTemplateService) Create(ctx context.Context, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	return m.createFn(ctx, tenantID, name, description, message, senderID, status)
}
func (m *testSMSTemplateService) Update(ctx context.Context, stUUID uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	return m.updateFn(ctx, stUUID, tenantID, name, description, message, senderID, status)
}
func (m *testSMSTemplateService) UpdateStatus(ctx context.Context, stUUID uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error) {
	return m.updateStatusFn(ctx, stUUID, tenantID, status)
}
func (m *testSMSTemplateService) Delete(ctx context.Context, stUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	return m.deleteFn(ctx, stUUID, tenantID)
}

type testLoginTemplateService struct {
	getAllFn       func(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error)
	getByUUIDFn    func(ctx context.Context, ltUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error)
	createFn       func(ctx context.Context, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	updateFn       func(ctx context.Context, ltUUID uuid.UUID, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	updateStatusFn func(ctx context.Context, ltUUID uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error)
	deleteFn       func(ctx context.Context, ltUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error)
}

func (m *testLoginTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, name, status, template, isDefault, isSystem, page, limit, sortBy, sortOrder)
}
func (m *testLoginTemplateService) GetByUUID(ctx context.Context, ltUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
	return m.getByUUIDFn(ctx, ltUUID, tenantID)
}
func (m *testLoginTemplateService) Create(ctx context.Context, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	return m.createFn(ctx, tenantID, name, description, template, metadata, status)
}
func (m *testLoginTemplateService) Update(ctx context.Context, ltUUID uuid.UUID, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	return m.updateFn(ctx, ltUUID, tenantID, name, description, template, metadata, status)
}
func (m *testLoginTemplateService) UpdateStatus(ctx context.Context, ltUUID uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error) {
	return m.updateStatusFn(ctx, ltUUID, tenantID, status)
}
func (m *testLoginTemplateService) Delete(ctx context.Context, ltUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
	return m.deleteFn(ctx, ltUUID, tenantID)
}
