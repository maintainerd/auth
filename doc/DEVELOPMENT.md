# Development Guide

This guide covers local development setup and troubleshooting for the maintainerd auth service.

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- Git

### Start Development Environment
```bash
# Start all services with hot reloading
./scripts/dev.sh start

# View logs
./scripts/dev.sh logs

# View logs for specific service
./scripts/dev.sh logs auth
```

## 🔄 Hot Reloading

The development environment uses [Air](https://github.com/air-verse/air) for automatic code reloading.

### How It Works
1. **File Watching**: Air watches for changes in `.go` files
2. **Automatic Rebuild**: When files change, Air rebuilds the binary
3. **Process Restart**: Air restarts the application with the new binary
4. **Fast Feedback**: Changes are reflected within 1-2 seconds

### Watched Files
- All `.go` files in the project
- Template files (`.tpl`, `.tmpl`, `.html`)
- Excludes: `tmp/`, `vendor/`, `internal/gen/`, `.git/`

## 🛠️ Development Commands

```bash
# Start development environment
./scripts/dev.sh start

# Stop development environment
./scripts/dev.sh stop

# Restart everything
./scripts/dev.sh restart

# Rebuild and restart auth service only
./scripts/dev.sh reload

# View service status
./scripts/dev.sh status

# Enter auth container shell
./scripts/dev.sh shell

# Clean up everything (containers, images, volumes)
./scripts/dev.sh clean
```

## 🐛 Troubleshooting

### Hot Reloading Not Working

#### 1. Check Air is Running
```bash
# View auth service logs
./scripts/dev.sh logs auth

# You should see Air startup messages like:
# [INFO] watching .
# [INFO] building...
```

#### 2. Check File Permissions
```bash
# Enter container shell
./scripts/dev.sh shell

# Check if files are mounted correctly
ls -la /usr/src/app

# Check Air process
ps aux | grep air
```

#### 3. Manual Restart
```bash
# If hot reload fails, manually restart auth service
./scripts/dev.sh reload
```

#### 4. Check Volume Mounts
Ensure your `docker-compose.yml` has correct volume mounts:
```yaml
volumes:
  - .:/usr/src/app  # Source code
  - go-mod-cache:/go/pkg/mod  # Go modules cache
  - go-build-cache:/root/.cache/go-build  # Build cache
```

### Build Errors

#### 1. Check Build Logs
```bash
# View detailed logs
./scripts/dev.sh logs auth

# Check build error log
docker-compose exec auth cat tmp/build-errors.log
```

#### 2. Clear Build Cache
```bash
# Clean and restart
./scripts/dev.sh clean
./scripts/dev.sh start
```

#### 3. Check Go Modules
```bash
# Enter container and check modules
./scripts/dev.sh shell
go mod tidy
go mod download
```

### Container Issues

#### 1. Container Won't Start
```bash
# Check container status
./scripts/dev.sh status

# View container logs
docker-compose logs auth

# Check Docker daemon
docker info
```

#### 2. Port Conflicts
```bash
# Check if ports are in use
netstat -tulpn | grep :8080
netstat -tulpn | grep :5433

# Stop conflicting services or change ports in docker-compose.yml
```

#### 3. Database Connection Issues
```bash
# Check database container
docker-compose logs postgres-db

# Test database connection from auth container
./scripts/dev.sh shell
nc -zv postgres-db 5432
```

### Performance Issues

#### 1. Slow Builds
- Ensure Go module cache is working: `go-mod-cache` volume
- Check available disk space
- Consider increasing Docker memory allocation

#### 2. Slow File Watching
- Reduce watched files in `.air.toml`
- Exclude unnecessary directories
- Check if antivirus is scanning Docker volumes

## 📁 Project Structure

```
.
├── cmd/server/          # Application entry point
├── internal/            # Internal packages
│   ├── app/            # Application setup
│   ├── config/         # Configuration
│   ├── handler/        # HTTP handlers
│   ├── service/        # Business logic
│   ├── repository/     # Data access
│   └── model/          # Data models
├── scripts/            # Development scripts
├── docker-compose.yml  # Development environment
├── Dockerfile.local    # Development Dockerfile
└── .air.toml          # Hot reload configuration
```

## 🔧 Configuration Files

### .air.toml
Controls hot reloading behavior:
- `delay`: Time to wait after file change (ms)
- `include_ext`: File extensions to watch
- `exclude_dir`: Directories to ignore

### docker-compose.yml
Development environment setup:
- Volume mounts for source code
- Environment variables
- Service dependencies

### Dockerfile.local
Development container setup:
- Go toolchain
- Air for hot reloading
- Development dependencies

## 📝 Best Practices

### 1. Code Changes
- Save files to trigger automatic rebuild
- Watch logs for build errors
- Use `./scripts/dev.sh reload` if hot reload fails

### 2. Database Changes
- Migrations run automatically on startup
- Use `./scripts/dev.sh restart` after schema changes
- Check migration logs: `./scripts/dev.sh logs auth`

### 3. Dependency Changes
- Restart after `go.mod` changes: `./scripts/dev.sh restart`
- Clear cache if needed: `./scripts/dev.sh clean`

### 4. Environment Variables
- Changes to `.env` require restart: `./scripts/dev.sh restart`
- Check environment in container: `./scripts/dev.sh shell` then `env`

## 🚨 Common Issues

### "Permission Denied" Errors
- Check file permissions on host
- Ensure Docker has access to project directory
- On Windows: Check Docker Desktop file sharing settings

### "Module Not Found" Errors
- Run `./scripts/dev.sh restart` to rebuild with fresh modules
- Check `go.mod` and `go.sum` are properly mounted

### "Port Already in Use" Errors
- Stop conflicting services
- Change ports in `docker-compose.yml`
- Use `./scripts/dev.sh clean` to remove old containers

### Air Not Detecting Changes
- Check `.air.toml` configuration
- Verify file extensions are included
- Ensure directories aren't excluded
- Try manual restart: `./scripts/dev.sh reload`
