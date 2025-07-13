## Packages

### Step 1 : Create the project
```bash
go mod init github.com/maintainerd/auth
```

### Step 2 : Install dependencies
```bash
go get -u github.com/gin-gonic/gin
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get github.com/joho/godotenv
go get github.com/google/uuid
go get google.golang.org/grpc@latest
# For generating grpc
go get google.golang.org/protobuf@latest
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go get google.golang.org/protobuf/cmd/protoc-gen-go@latest
# Adding sub module
git submodule add https://github.com/maintainerd/contract.git internal/contract
git submodule update --init --recursive
# Re-adding sub module
git submodule deinit -f internal/contract
rm -rf .git/modules/internal/contract
rm -rf internal/contract
git rm -f internal/contract
# Removing sub module
git rm -f internal/contract
git config --remove-section submodule.internal/contract
rm -rf internal/contract
rm -rf .git/modules/internal/contract
git commit -am "Remove submodule internal/contract"
```

### Step 3: Install CLI
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Commands
```bash
# Create migration files
goose -dir db/migrations create create_tasks_table sql

# Run app
go run cmd/main.go
make run

# Run migrations
make migrate-up
make migrate-down

# Run seeder
make seed-local
```

