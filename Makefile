.PHONY: templ generate build run dev test migrate-up migrate-down install sqlc clean docker-build docker-run docker-stop docker-clean docker-logs prod-up prod-down prod-logs prod-update prod-backup

templ:
	@echo "Generating templ files..."
	@templ generate

generate: templ sqlc
	@echo "All code generated"

build: generate
	go build -o bin/cleaning-scheduler cmd/server/main.go

run: build
	./bin/cleaning-scheduler

dev:
	air

test:
	go test -v ./...

migrate-up:
	sqlite3 cleaning.db < internal/database/migrations/001_initial_schema.sql

migrate-down:
	rm -f cleaning.db

install:
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/air-verse/air@latest
	~/go/bin/sqlc generate
	make migrate-up

sqlc:
	~/go/bin/sqlc generate

clean:
	rm -rf bin/
	rm -f cleaning.db

seed: migrate-up
	@echo "Database initialized. Run 'make run' and import the CSV file."

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t cleaning-scheduler:latest .

docker-run:
	@echo "Starting containers with docker-compose..."
	docker-compose up -d
	@echo "Application running at http://localhost:8080"

docker-stop:
	@echo "Stopping containers..."
	docker-compose down

docker-clean:
	@echo "Cleaning up containers and volumes..."
	docker-compose down -v
	docker rmi cleaning-scheduler:latest || true

docker-logs:
	docker-compose logs -f

# Production deployment targets
prod-up:
	@echo "Starting production deployment with Caddy..."
	docker-compose -f docker-compose.prod.yml up -d
	@echo "Application running at https://your-domain.com"

prod-down:
	@echo "Stopping production deployment..."
	docker-compose -f docker-compose.prod.yml down

prod-logs:
	docker-compose -f docker-compose.prod.yml logs -f

prod-update:
	@echo "Updating to latest version..."
	docker pull druarnfield/cleaning-scheduler:latest
	docker-compose -f docker-compose.prod.yml down
	docker-compose -f docker-compose.prod.yml up -d
	@echo "Update complete!"

prod-backup:
	@echo "Creating database backup..."
	@mkdir -p backups
	docker run --rm \
		-v cleaning-scheduler_cleaning_data:/data \
		-v $(PWD)/backups:/backup \
		alpine cp /data/cleaning.db /backup/cleaning-$$(date +%Y%m%d_%H%M%S).db
	@echo "Backup created in ./backups/"