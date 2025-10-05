#!/bin/bash

# Development helper script for maintainerd auth service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
}

# Function to check if docker-compose is available
check_docker_compose() {
    if ! command -v docker-compose > /dev/null 2>&1; then
        print_error "docker-compose is not installed or not in PATH."
        exit 1
    fi
}

# Function to start development environment
dev_start() {
    print_status "Starting development environment..."
    check_docker
    check_docker_compose
    
    # Build and start services
    docker-compose up --build -d
    
    print_success "Development environment started!"
    print_status "Services running:"
    print_status "  - Auth API: http://localhost:8080"
    print_status "  - Nginx Proxy: http://localhost:80"
    print_status "  - PostgreSQL: localhost:5433"
    print_status "  - Redis: localhost:6379"
    print_status "  - RabbitMQ Management: http://localhost:15672"
    print_status ""
    print_status "To view logs: ./scripts/dev.sh logs"
    print_status "To stop: ./scripts/dev.sh stop"
}

# Function to stop development environment
dev_stop() {
    print_status "Stopping development environment..."
    docker-compose down
    print_success "Development environment stopped!"
}

# Function to restart development environment
dev_restart() {
    print_status "Restarting development environment..."
    docker-compose down
    docker-compose up --build -d
    print_success "Development environment restarted!"
}

# Function to view logs
dev_logs() {
    if [ -n "$2" ]; then
        print_status "Showing logs for service: $2"
        docker-compose logs -f "$2"
    else
        print_status "Showing logs for all services (Ctrl+C to exit)"
        docker-compose logs -f
    fi
}

# Function to rebuild and restart auth service only
dev_reload() {
    print_status "Rebuilding and restarting auth service..."
    docker-compose up --build -d auth
    print_success "Auth service reloaded!"
}

# Function to show status
dev_status() {
    print_status "Development environment status:"
    docker-compose ps
}

# Function to clean up
dev_clean() {
    print_warning "This will remove all containers, images, and volumes!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Cleaning up development environment..."
        docker-compose down -v --rmi all
        docker system prune -f
        print_success "Development environment cleaned!"
    else
        print_status "Clean up cancelled."
    fi
}

# Function to enter auth container shell
dev_shell() {
    print_status "Entering auth container shell..."
    docker-compose exec auth sh
}

# Function to show help
show_help() {
    echo "Development helper script for maintainerd auth service"
    echo ""
    echo "Usage: ./scripts/dev.sh [command]"
    echo ""
    echo "Commands:"
    echo "  start     Start development environment"
    echo "  stop      Stop development environment"
    echo "  restart   Restart development environment"
    echo "  reload    Rebuild and restart auth service only"
    echo "  logs      Show logs (optional: specify service name)"
    echo "  status    Show status of all services"
    echo "  shell     Enter auth container shell"
    echo "  clean     Clean up all containers, images, and volumes"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./scripts/dev.sh start"
    echo "  ./scripts/dev.sh logs auth"
    echo "  ./scripts/dev.sh reload"
}

# Main script logic
case "${1:-help}" in
    start)
        dev_start
        ;;
    stop)
        dev_stop
        ;;
    restart)
        dev_restart
        ;;
    reload)
        dev_reload
        ;;
    logs)
        dev_logs "$@"
        ;;
    status)
        dev_status
        ;;
    shell)
        dev_shell
        ;;
    clean)
        dev_clean
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
