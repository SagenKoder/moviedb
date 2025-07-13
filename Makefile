.PHONY: dev build clean install test lint typecheck

# Development commands
dev-frontend:
	cd web && npm run dev

dev-backend:
	export $$(grep -v '^#' .env | xargs) && go run cmd/server/main.go

dev:
	# Run both frontend and backend in development
	make -j2 dev-frontend dev-backend

# Build commands
build-frontend:
	cd web && npm run build

build-backend:
	go build -o bin/moviedb ./cmd/server

build: build-frontend build-backend

# Install dependencies
install-frontend:
	cd web && npm install

install-backend:
	go mod download

install: install-backend install-frontend

# Testing and linting
test-backend:
	go test ./...

test-frontend:
	cd web && npm test

test: test-backend test-frontend

lint-backend:
	go vet ./...
	go fmt ./...

lint-frontend:
	cd web && npm run lint

lint: lint-backend lint-frontend

typecheck-frontend:
	cd web && npm run typecheck

typecheck: typecheck-frontend

# Database
db-reset:
	rm -f moviedb.db

# Clean build artifacts
clean:
	rm -rf web/dist bin/ moviedb.db

# Production build with embedded static files
# Note: Frontend env vars (VITE_*) are read from web/.env.local or web/.env.production during build
build-prod: build-frontend
	go build -ldflags="-s -w" -o bin/moviedb ./cmd/server

# Docker build
docker-build:
	docker build -t moviedb .

# Help
help:
	@echo "Available commands:"
	@echo "  dev              - Run both frontend and backend in development mode"
	@echo "  dev-frontend     - Run frontend development server"
	@echo "  dev-backend      - Run backend development server"
	@echo "  build            - Build both frontend and backend"
	@echo "  build-frontend   - Build frontend for production"
	@echo "  build-backend    - Build backend binary"
	@echo "  build-prod       - Build optimized production binary"
	@echo "  install          - Install all dependencies"
	@echo "  test             - Run all tests"
	@echo "  lint             - Run all linters"
	@echo "  typecheck        - Run TypeScript type checking"
	@echo "  clean            - Clean build artifacts and database"
	@echo "  db-reset         - Reset database (delete moviedb.db)"
	@echo "  docker-build     - Build Docker image"