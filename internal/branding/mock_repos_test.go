package branding

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: BrandingRepository
// ---------------------------------------------------------------------------

type mockBrandingRepo struct {
	findByTenantIDFn func(int64) (*Branding, error)
	createFn         func(*Branding) (*Branding, error)
	createOrUpdateFn func(*Branding) (*Branding, error)
}

func (m *mockBrandingRepo) WithTx(_ *gorm.DB) BrandingRepository { return m }

func (m *mockBrandingRepo) FindByTenantID(tenantID int64) (*Branding, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockBrandingRepo) Create(e *Branding) (*Branding, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockBrandingRepo) CreateOrUpdate(e *Branding) (*Branding, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockBrandingRepo) FindAll(p ...string) ([]Branding, error)              { return nil, nil }
func (m *mockBrandingRepo) FindByUUID(id any, p ...string) (*Branding, error)    { return nil, nil }
func (m *mockBrandingRepo) FindByUUIDs(ids []string, p ...string) ([]Branding, error) {
	return nil, nil
}
func (m *mockBrandingRepo) FindByID(id any, p ...string) (*Branding, error) { return nil, nil }
func (m *mockBrandingRepo) UpdateByUUID(id, data any) (*Branding, error)    { return nil, nil }
func (m *mockBrandingRepo) UpdateByID(id, data any) (*Branding, error)      { return nil, nil }
func (m *mockBrandingRepo) DeleteByUUID(id any) error                       { return nil }
func (m *mockBrandingRepo) DeleteByID(id any) error                         { return nil }
func (m *mockBrandingRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Branding], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: EmailTemplateRepository
// ---------------------------------------------------------------------------

type mockEmailTemplateRepo struct {
	findByUUIDAndTenantIDFn func(uuid.UUID, int64, ...string) (*EmailTemplate, error)
	findPaginatedFn         func(EmailTemplateRepositoryGetFilter) (*PaginationResult[EmailTemplate], error)
	createFn                func(*EmailTemplate) (*EmailTemplate, error)
	updateByUUIDFn          func(any, any) (*EmailTemplate, error)
	deleteByUUIDFn          func(any) error
	findByNameFn            func(string) (*EmailTemplate, error)
}

func (m *mockEmailTemplateRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*EmailTemplate, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID, p...)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindPaginated(f EmailTemplateRepositoryGetFilter) (*PaginationResult[EmailTemplate], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[EmailTemplate]{}, nil
}
func (m *mockEmailTemplateRepo) Create(e *EmailTemplate) (*EmailTemplate, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockEmailTemplateRepo) CreateOrUpdate(e *EmailTemplate) (*EmailTemplate, error) {
	return e, nil
}
func (m *mockEmailTemplateRepo) FindAll(p ...string) ([]EmailTemplate, error) { return nil, nil }
func (m *mockEmailTemplateRepo) FindByUUID(id any, p ...string) (*EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByID(id any, p ...string) (*EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByUUID(id, data any) (*EmailTemplate, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByID(id, data any) (*EmailTemplate, error) { return nil, nil }
func (m *mockEmailTemplateRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockEmailTemplateRepo) DeleteByID(id any) error { return nil }
func (m *mockEmailTemplateRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[EmailTemplate], error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByName(name string) (*EmailTemplate, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: SMSTemplateRepository
// ---------------------------------------------------------------------------

type mockSMSTemplateRepo struct {
	findByUUIDAndTenantIDFn func(string, int64) (*SMSTemplate, error)
	findPaginatedFn         func(SMSTemplateRepositoryGetFilter) (*PaginationResult[SMSTemplate], error)
	createFn                func(*SMSTemplate) (*SMSTemplate, error)
	updateByUUIDFn          func(any, any) (*SMSTemplate, error)
	deleteByUUIDFn          func(any) error
}

func (m *mockSMSTemplateRepo) FindByUUIDAndTenantID(id string, tenantID int64) (*SMSTemplate, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID)
	}
	return nil, nil
}
func (m *mockSMSTemplateRepo) FindPaginated(f SMSTemplateRepositoryGetFilter) (*PaginationResult[SMSTemplate], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[SMSTemplate]{}, nil
}
func (m *mockSMSTemplateRepo) Create(e *SMSTemplate) (*SMSTemplate, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockSMSTemplateRepo) CreateOrUpdate(e *SMSTemplate) (*SMSTemplate, error) { return e, nil }
func (m *mockSMSTemplateRepo) FindAll(p ...string) ([]SMSTemplate, error)          { return nil, nil }
func (m *mockSMSTemplateRepo) FindByUUID(id any, p ...string) (*SMSTemplate, error) {
	return nil, nil
}
func (m *mockSMSTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]SMSTemplate, error) {
	return nil, nil
}
func (m *mockSMSTemplateRepo) FindByID(id any, p ...string) (*SMSTemplate, error) { return nil, nil }
func (m *mockSMSTemplateRepo) UpdateByUUID(id, data any) (*SMSTemplate, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockSMSTemplateRepo) UpdateByID(id, data any) (*SMSTemplate, error) { return nil, nil }
func (m *mockSMSTemplateRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockSMSTemplateRepo) DeleteByID(id any) error { return nil }
func (m *mockSMSTemplateRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[SMSTemplate], error) {
	return nil, nil
}
func (m *mockSMSTemplateRepo) WithTx(_ *gorm.DB) SMSTemplateRepository { return m }
func (m *mockSMSTemplateRepo) FindByName(_ string) (*SMSTemplate, error) { return nil, nil }

// ---------------------------------------------------------------------------
// Mock: LoginTemplateRepository
// ---------------------------------------------------------------------------

type mockLoginTemplateRepo struct {
	findByUUIDAndTenantIDFn func(uuid.UUID, int64, ...string) (*LoginTemplate, error)
	findPaginatedFn         func(LoginTemplateRepositoryGetFilter) (*PaginationResult[LoginTemplate], error)
	createFn                func(*LoginTemplate) (*LoginTemplate, error)
	updateByUUIDFn          func(any, any) (*LoginTemplate, error)
	deleteByUUIDFn          func(any) error
}

func (m *mockLoginTemplateRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*LoginTemplate, error) {
	if m.findByUUIDAndTenantIDFn != nil {
		return m.findByUUIDAndTenantIDFn(id, tenantID, p...)
	}
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindPaginated(f LoginTemplateRepositoryGetFilter) (*PaginationResult[LoginTemplate], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[LoginTemplate]{}, nil
}
func (m *mockLoginTemplateRepo) Create(e *LoginTemplate) (*LoginTemplate, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockLoginTemplateRepo) CreateOrUpdate(e *LoginTemplate) (*LoginTemplate, error) {
	return e, nil
}
func (m *mockLoginTemplateRepo) FindAll(p ...string) ([]LoginTemplate, error) { return nil, nil }
func (m *mockLoginTemplateRepo) FindByUUID(id any, p ...string) (*LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindByID(id any, p ...string) (*LoginTemplate, error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) UpdateByUUID(id, data any) (*LoginTemplate, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	return nil, nil
}
func (m *mockLoginTemplateRepo) UpdateByID(id, data any) (*LoginTemplate, error) { return nil, nil }
func (m *mockLoginTemplateRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockLoginTemplateRepo) DeleteByID(id any) error { return nil }
func (m *mockLoginTemplateRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[LoginTemplate], error) {
	return nil, nil
}
func (m *mockLoginTemplateRepo) FindByName(_ string) (*LoginTemplate, error) { return nil, nil }
