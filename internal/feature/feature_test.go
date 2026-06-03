package feature

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockReader struct {
	setting *Setting
	err     error
	called  bool
	tenant  int64
}

func (m *mockReader) FindByTenantID(tenantID int64) (*Setting, error) {
	m.called = true
	m.tenant = tenantID
	return m.setting, m.err
}

func TestEnabled(t *testing.T) {
	tests := []struct {
		name       string
		reader     *mockReader
		tenantID   int64
		key        string
		defaultVal bool
		want       bool
		wantCalled bool
		wantTenant int64
	}{
		{
			name:       "nil reader returns default",
			tenantID:   1,
			key:        "new_flow",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "zero tenant returns default",
			reader:     &mockReader{},
			tenantID:   0,
			key:        "new_flow",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "blank key returns default",
			reader:     &mockReader{},
			tenantID:   1,
			key:        "   ",
			defaultVal: false,
			want:       false,
		},
		{
			name:       "reader error returns default",
			reader:     &mockReader{err: errors.New("db error")},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: true,
			want:       true,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "nil setting returns default",
			reader:     &mockReader{},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: false,
			want:       false,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "empty feature flags returns default",
			reader:     &mockReader{setting: &Setting{}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: true,
			want:       true,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "malformed json returns default",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{bad`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: false,
			want:       false,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "missing key returns default",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"other":true}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: false,
			want:       false,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "bool true returns true",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"new_flow":true}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: false,
			want:       true,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "bool false returns false",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"new_flow":false}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: true,
			want:       false,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "string true returns true",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"new_flow":"true"}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: false,
			want:       true,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "string false returns false",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"new_flow":"false"}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: true,
			want:       false,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "invalid string returns default",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"new_flow":"sometimes"}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: true,
			want:       true,
			wantCalled: true,
			wantTenant: 7,
		},
		{
			name:       "unsupported type returns default",
			reader:     &mockReader{setting: &Setting{FeatureFlags: []byte(`{"new_flow":1}`)}},
			tenantID:   7,
			key:        "new_flow",
			defaultVal: false,
			want:       false,
			wantCalled: true,
			wantTenant: 7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reader Reader
			if tc.reader != nil {
				reader = tc.reader
			}

			got := Enabled(reader, tc.tenantID, tc.key, tc.defaultVal)

			assert.Equal(t, tc.want, got)
			if tc.reader != nil {
				assert.Equal(t, tc.wantCalled, tc.reader.called)
				assert.Equal(t, tc.wantTenant, tc.reader.tenant)
			}
		})
	}
}
