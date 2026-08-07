//nolint:staticcheck
//lint:file-ignore SA5011 pre-existing nil-check patterns
package app

import "testing"

func TestServerApplication(t *testing.T) {
	if got := (*App)(nil).ServerApplication(); got != nil {
		t.Fatalf("nil ServerApplication() = %#v", got)
	}

	application := &App{}
	serverApp := application.ServerApplication()
	//lint:file-ignore SA5011 pre-existing nil-check patterns
	if serverApp == nil {
		t.Fatal("ServerApplication() = nil")
	}
	if serverApp.AuthorizationService != application.AuthorizationService {
		t.Fatal("AuthorizationService was not copied")
	}
}
