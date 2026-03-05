.PHONY: dev build up down test clean

# Local development: NATS in Docker, Go + UI dev servers locally
dev:
	docker compose up nats -d
	@echo "NATS running on :4222"
	@echo "Starting Go backend..."
	go run ./cmd/semsage -config configs/semsage.json &
	@echo "Starting UI dev server..."
	cd ui && npm run dev

# Build Go binary and UI static assets
build:
	go build -o bin/semsage ./cmd/semsage
	cd ui && npm run build

# Build and start all services via Docker Compose
up: build
	docker compose up --build -d

# Stop all Docker Compose services
down:
	docker compose down

# Run all tests
test:
	go test ./...
	cd ui && npm run test:e2e

# Remove build artifacts
clean:
	rm -rf bin/
	rm -rf ui/build/
