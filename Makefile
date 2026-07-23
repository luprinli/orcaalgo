.PHONY: all build build-go build-python install-python test test-go test-python test-related test-related-go test-related-python test-guardian lint lint-go lint-python lint-web clean docker-build docker-up docker-down preflight anti-pattern coverage parity-scan

GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%FT%TZ)
LDFLAGS    := -X github.com/lee-econ/orca-core/internal/version.Commit=$(GIT_COMMIT) -X github.com/lee-econ/orca-core/internal/version.BuildTime=$(BUILD_TIME)

all: install-python build

install-python:
	@pip install -e ".[dev]"

build-go:
	@go build -ldflags "$(LDFLAGS)" -o bin/orca-server.exe ./cmd/orca-server
	@go build -ldflags "$(LDFLAGS)" -o bin/orca-cli.exe ./cmd/orca-cli

build-python:
	@python -c "from orca.ir.loader import load_ir; print('orca package OK')"

build: build-go build-python

test-go:
	@go test ./internal/... -v -count=1

test-python:
	@cd tests && python -m pytest . -v

test: test-go test-python

test-related:
	@python scripts/test_related.py

test-related-go:
	@python scripts/test_related.py --language go

test-related-python:
	@python scripts/test_related.py --language python

test-guardian:
	@echo "=== Guardian Python Tests ==="
	@cd tests && python -m pytest guardian/ -v --tb=short
	@echo "=== Guardian Go Tests ==="
	@go test -tags=guardian ./tests/guardian/ -v -count=1

lint-go:
	@golangci-lint run ./...

lint-python:
	@ruff check orca/ tests/
	@mypy orca/

lint: lint-go lint-python

lint-web:
	@cd web && npx eslint src/ --max-warnings 0

clean:
	@rm -rf bin/ .pytest_cache/ orca/__pycache__/ tests/__pycache__/ .mypy_cache/ .ruff_cache/ reports/ 2>/dev/null || true
	@if [ "$$OS" = "Windows_NT" ]; then cmd /c "if exist bin rmdir /s /q bin 2>nul"; fi

docker-build:
	docker build -t orca-core:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

anti-pattern:
	@python scripts/anti_pattern_scan.py

preflight:
	@orca preflight --strict

parity-scan:
	@python scripts/parity_drift_scan.py --strict

coverage:
	@echo "=== Python Coverage ==="
	@cd tests && python -m pytest . -v --cov=orca --cov-report=html --cov-report=term
	@echo "=== Go Coverage ==="
	@go test ./internal/... -coverprofile=coverage.out -coverpkg=./internal/...
	@go tool cover -func=coverage.out | grep total
