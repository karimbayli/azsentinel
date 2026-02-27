# Sentinel V2

# Build
.PHONY: build build-central build-probe test lint clean docker-central docker-probe

VERSION := 1.0.0
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

build: build-central build-probe

build-central:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/sentinel-central ./cmd/central/

build-probe:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/sentinel-probe ./cmd/probe-agent/

# Test
test:
	go test -v -race -count=1 ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	go vet ./...
	@which staticcheck > /dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

fmt:
	gofmt -w .

# Docker
docker-central:
	docker build -f Dockerfile.central -t sentinel-central:$(VERSION) .

docker-probe:
	docker build -f Dockerfile.probe -t sentinel-probe:$(VERSION) .

docker-all: docker-central docker-probe

# Deploy
deploy-central:
	bash scripts/deploy-central.sh

deploy-node:
	@echo "Usage: make deploy-node NODE_ID=node-eu REGION=eu-frankfurt COUNTRY=DE CENTRAL_URL=https://sentinel.example.com"
	bash scripts/deploy-node.sh $(NODE_ID) $(REGION) $(COUNTRY) $(CENTRAL_URL)

# Database
db-backup:
	bash scripts/backup-db.sh

# Local development
dev-central: build-central
	./bin/sentinel-central --config configs/central.example.yaml

dev-probe: build-probe
	./bin/sentinel-probe --config configs/probe.example.yaml

dev-up:
	docker compose -f deployments/docker-compose.central.yml up -d

dev-down:
	docker compose -f deployments/docker-compose.central.yml down

dev-logs:
	docker compose -f deployments/docker-compose.central.yml logs -f

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html

# Help
help:
	@echo "Sentinel V2 — Azerbaijan Internet Infrastructure Monitor"
	@echo ""
	@echo "Build:"
	@echo "  make build           Build both binaries"
	@echo "  make build-central   Build central server"
	@echo "  make build-probe     Build probe agent"
	@echo ""
	@echo "Test:"
	@echo "  make test            Run all tests"
	@echo "  make test-coverage   Run with coverage report"
	@echo "  make lint            Run go vet + staticcheck"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-all      Build all Docker images"
	@echo "  make dev-up          Start dev environment"
	@echo "  make dev-down        Stop dev environment"
	@echo ""
	@echo "Deploy:"
	@echo "  make deploy-central  Deploy central server"
	@echo "  make deploy-node     Deploy probe node"
