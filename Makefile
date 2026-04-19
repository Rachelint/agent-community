.PHONY: help deps dev server web build clean tidy

help:
	@echo "Targets:"
	@echo "  deps     Install web deps"
	@echo "  dev      Run server + web in parallel"
	@echo "  server   Run Go server only"
	@echo "  web      Run Vite dev server only"
	@echo "  build    Build single binary with embedded UI"
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
	CGO_ENABLED=0 go build -o bin/agent-community ./cmd/server

tidy:
	go mod tidy

clean:
	rm -rf bin web/dist web/node_modules
