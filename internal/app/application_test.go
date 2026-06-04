package app

import "testing"

func TestServerApplication(t *testing.T) {
	if got := (*App)(nil).ServerApplication(); got != nil {
		t.Fatalf("nil ServerApplication() = %#v", got)
	}

	application := &App{}
	serverApp := application.ServerApplication()
	if serverApp == nil {
		t.Fatal("ServerApplication() = nil")
	}
	if serverApp.AuthorizationService != application.AuthorizationService {
		t.Fatal("AuthorizationService was not copied")
	}
}
