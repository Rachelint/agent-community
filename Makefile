.PHONY: help deps dev server web build test check clean tidy

help:
	@echo "Targets:"
	@echo "  deps     Install web deps"
	@echo "  dev      Run server + web in parallel"
	@echo "  server   Run Go server only"
	@echo "  web      Run Vite dev server only"
	@echo "  build    Build single binary with embedded UI"
	@echo "  test     Run Go tests"
	@echo "  check    Run formatting, tests, vet, and web typecheck"
	@echo "  tidy     go mod tidy"
	@echo "  clean    Remove build artifacts"

deps:
	cd web && npm install

dev:
	./scripts/dev.sh

server:
	go run ./cmd/server

web:
	cd web && npm run dev

build:
	cd web && npm run build
	rm -rf cmd/server/ui && cp -r web/dist cmd/server/ui
	CGO_ENABLED=0 go build -o bin/agent-community ./cmd/server

test:
	go test ./...

check:
	@test -z "$$(gofmt -l $$(git ls-files '*.go'))"
	go test ./...
	go vet ./...
	cd web && npm run typecheck

tidy:
	go mod tidy

clean:
	rm -rf bin web/dist web/node_modules
	cd cmd/server/ui && find . ! -name '.gitkeep' -type f -delete 2>/dev/null; true
