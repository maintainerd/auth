run:
	go run cmd/main.go

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-down-all:
	./scripts/migrate.sh down-to 0

migrate-reset:
	./scripts/migrate.sh reset

seed-local:
	go run db/seed/task_seeder.go
