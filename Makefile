run:
	go run cmd/main.go

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

seed-local:
	go run db/seed/task_seeder.go
