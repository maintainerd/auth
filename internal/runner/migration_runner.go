package runner

import (
	"log"
	"os"
	"os/exec"
)

func RunMigrations(connectionString string) {
	log.Println("🏃 Running migrations...")

	// Migration files
	directories := []string{
		"migration/core",
		"migration/auth",
	}

	// Run migrations
	for _, dir := range directories {
		runMigrationDir(dir, connectionString)
	}

	log.Println("✅ Migrations completed.")
}

func runMigrationDir(dir string, connectionString string) {
	dbPrefix := os.Getenv("DB_TABLE_PREFIX")
	cmd := exec.Command("goose", "-dir", dir, "postgres", connectionString, "up")
	cmd.Env = append(os.Environ(), "DB_TABLE_PREFIX="+dbPrefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("❌ Migration failed in %s: %v\nOutput: %s", dir, err, string(output))
	}
	log.Printf("✅ Migrations applied from %s", dir)
}
