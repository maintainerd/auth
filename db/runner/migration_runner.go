package runner

import (
	"log"
	"os/exec"
)

func RunDefaultMigrations(connectionString string) {
	log.Println("🏃 Running default migrations...")
	runMigrationDir("./db/migration/v1", connectionString)
	log.Println("✅ Default migrations completed.")
}

func runMigrationDir(dir string, connectionString string) {
	cmd := exec.Command("goose", "-dir", dir, "postgres", connectionString, "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("❌ Migration failed in %s: %v\nOutput: %s", dir, err, string(output))
	}
	log.Printf("✅ Migrations applied from %s", dir)
}
