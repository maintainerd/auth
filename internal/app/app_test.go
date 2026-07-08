package app

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

func TestNewAppWiresAllExportedServices(t *testing.T) {
	origPublicHostname := config.AppPublicHostname
	config.AppPublicHostname = "http://localhost:8081"
	t.Cleanup(func() {
		config.AppPublicHostname = origPublicHostname
	})

	application, err := NewApp(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}
	value := reflect.ValueOf(application).Elem()
	typ := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" || field.Name == "DB" || field.Name == "RedisClient" || field.Name == "Cache" {
			continue
		}
		if value.Field(i).IsNil() {
			t.Fatalf("%s was not wired", field.Name)
		}
	}
}

func TestNewApp_ServiceInitError(t *testing.T) {
	origPublicHostname := config.AppPublicHostname
	config.AppPublicHostname = "not a url with spaces"
	t.Cleanup(func() {
		config.AppPublicHostname = origPublicHostname
	})

	application, err := NewApp(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("NewApp error = nil")
	}
	if application != nil {
		t.Fatalf("NewApp application = %#v", application)
	}
	if !strings.Contains(err.Error(), "service init failed") {
		t.Fatalf("NewApp error = %v", err)
	}
}
