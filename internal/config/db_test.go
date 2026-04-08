package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDBConnectionString(t *testing.T) {
	origHost := DBHost
	origPort := DBPort
	origUser := DBUser
	origPass := DBPassword
	origName := DBName
	origSSL := DBSSLMode
	t.Cleanup(func() {
		DBHost = origHost
		DBPort = origPort
		DBUser = origUser
		DBPassword = origPass
		DBName = origName
		DBSSLMode = origSSL
	})

	DBHost = "db.example.com"
	DBPort = "5432"
	DBUser = "admin"
	DBPassword = "s3cret"
	DBName = "mydb"
	DBSSLMode = "require"

	got := GetDBConnectionString()
	assert.Equal(t, "host=db.example.com port=5432 user=admin password=s3cret dbname=mydb sslmode=require", got)
}
