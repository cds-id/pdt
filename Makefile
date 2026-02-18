# PDT - Personal Development Tracker
# Makefile for common development tasks

# Colors
YELLOW := \033[0;33m
NC := \033[0m # No Color

# Default target
help:
	@echo ""
	@echo "$(YELLOW)PDT - Personal Development Tracker$(NC)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Frontend:"
	@echo "  frontend-install    Install frontend dependencies"
	@echo "  frontend           Run frontend development server"
	@echo "  frontend-build     Build frontend for production"
	@echo "  frontend-lint      Lint frontend code"
	@echo ""
	@echo "Backend:"
	@echo "  backend-install    Install backend dependencies"
	@echo "  backend-run        Run backend development server"
	@echo "  backend-build      Build backend for production"
	@echo ""
	@echo "Development:"
	@echo "  dev               Run both frontend and backend"
	@echo "  install           Install all dependencies"
	@echo "  build             Build all (frontend + backend)"
	@echo ""
	@echo "Database:"
	@echo "  db-migrate        Run database migrations"
	@echo "  db-seed           Seed database with sample data"
	@echo ""

# Frontend targets
frontend-install:
	@echo "$(YELLOW)Installing frontend dependencies...$(NC)"
	cd frontend && npm install

frontend:
	@echo "$(YELLOW)Starting frontend development server...$(NC)"
	cd frontend && npm run dev

frontend-build:
	@echo "$(YELLOW)Building frontend...$(NC)"
	cd frontend && npm run build

frontend-lint:
	@echo "$(YELLOW)Linting frontend code...$(NC)"
	cd frontend && npm run lint

# Backend targets
backend-install:
	@echo "$(YELLOW)Installing backend dependencies...$(NC)"
	cd backend && go mod download

backend-run:
	@echo "$(YELLOW)Starting backend development server...$(NC)"
	cd backend && go run cmd/server/main.go

backend-build:
	@echo "$(YELLOW)Building backend...$(NC)"
	cd backend && go build -o bin/server cmd/server/main.go

# Development targets
dev:
	@echo "$(YELLOW)Starting development environment...$(NC)"
	@echo "$(YELLOW)Note: Run 'backend-run' and 'frontend' in separate terminals$(NC)"
	make frontend

install: frontend-install backend-install

build: frontend-build backend-build

# Database targets
db-migrate:
	@echo "$(YELLOW)Running database migrations...$(NC)"
	cd backend && go run cmd/migrate/main.go

db-seed:
	@echo "$(YELLOW)Seeding database...$(NC)"
	cd backend && go run cmd/seed/main.go
