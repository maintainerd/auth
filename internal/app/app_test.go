package app

import (
	"reflect"
	"testing"

	"github.com/maintainerd/auth/internal/platform/config"
)

func TestNewAppWiresAllExportedServices(t *testing.T) {
	origPublicHostname := config.AppPublicHostname
	config.AppPublicHostname = "http://localhost:8081"
	t.Cleanup(func() {
		config.AppPublicHostname = origPublicHostname
	})

	application, err := NewApp(nil, nil)
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
