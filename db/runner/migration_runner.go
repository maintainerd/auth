package runner

import (
	"fmt"
	"log"
	"os/exec"
)

func RunDefaultMigrations(targetVersion string, connectionString string) {
	log.Println("🗄️ Running default migrations...")

	dir := fmt.Sprintf("./db/migration/%s/default", targetVersion)
	runMigrationDir(dir, connectionString)

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
