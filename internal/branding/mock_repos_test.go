package branding

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: BrandingRepository
// ---------------------------------------------------------------------------

type mockBrandingRepo struct {
	findByTenantIDFn    func(int64) (*Branding, error)
	findByUUIDFn        func(uuid.UUID) (*Branding, error)
	findByIDFn          func(any) (*Branding, error)
	findAllByTenantIDFn func(int64) ([]Branding, error)
	createFn            func(*Branding) (*Branding, error)
	createOrUpdateFn    func(*Branding) (*Branding, error)
	findActiveFn        func(int64) (*Branding, error)
	findSystemFn        func(int64) (*Branding, error)
	deactivateAllFn     func(int64) error
}

func (m *mockBrandingRepo) WithTx(_ *gorm.DB) BrandingRepository { return m }

func (m *mockBrandingRepo) FindByTenantID(tenantID int64) (*Branding, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockBrandingRepo) FindAllByTenantID(tenantID int64) ([]Branding, error) {
	if m.findAllByTenantIDFn != nil {
		return m.findAllByTenantIDFn(tenantID)
	}
	return nil, nil
}
func (m *mockBrandingRepo) Create(e *Branding) (*Branding, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockBrandingRepo) FindActive(tenantID int64) (*Branding, error) {
	if m.findActiveFn != nil {
		return m.findActiveFn(tenantID)
	}
	return nil, nil
}
func (m *mockBrandingRepo) FindSystem(tenantID int64) (*Branding, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn(tenantID)
	}
	return nil, nil
}
func (m *mockBrandingRepo) FindSystemDefault() (*Branding, error) {
	return m.FindSystem(0)
}
func (m *mockBrandingRepo) DeactivateAll(tenantID int64) error {
	if m.deactivateAllFn != nil {
		return m.deactivateAllFn(tenantID)
	}
	return nil
}
func (m *mockBrandingRepo) CreateOrUpdate(e *Branding) (*Branding, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(e)
	}
	return e, nil
}
func (m *mockBrandingRepo) FindAll(p ...string) ([]Branding, error) { return nil, nil }
func (m *mockBrandingRepo) FindByUUID(id any, p ...string) (*Branding, error) {
	if m.findByUUIDFn != nil {
		brandingUUID, _ := id.(uuid.UUID)
		return m.findByUUIDFn(brandingUUID)
	}
	return nil, nil
}
func (m *mockBrandingRepo) FindByUUIDs(ids []string, p ...string) ([]Branding, error) {
	return nil, nil
}
func (m *mockBrandingRepo) FindByID(id any, p ...string) (*Branding, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

// FindPublicByID shares findByIDFn: the mock has no columns, so the only thing
// the two differ in — whether logo bytes come back — is not observable here.
// The exclusion is a property of the real SQL, asserted separately.
func (m *mockBrandingRepo) FindPublicByID(id int64) (*Branding, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}
func (m *mockBrandingRepo) UpdateByUUID(id, data any) (*Branding, error) { return nil, nil }
func (m *mockBrandingRepo) UpdateByID(id, data any) (*Branding, error)   { return nil, nil }
func (m *mockBrandingRepo) DeleteByUUID(id any) error                    { return nil }
func (m *mockBrandingRepo) DeleteByID(id any) error                      { return nil }
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
func (m *mockEmailTemplateRepo) FindByNameAndTenantID(name string, tenantID int64) (*EmailTemplate, error) {
	return m.FindByName(name)
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

// ---------------------------------------------------------------------------
