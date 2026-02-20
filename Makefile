.PHONY: dev dev-server dev-web build build-server build-web build-all test test-server test-web test-e2e lint clean setup swagger gen-api

GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.Version=$(GIT_VERSION)

dev:
	@cd server && go run -ldflags='$(LDFLAGS)' cmd/server/main.go & \
	cd web && bun dev

dev-server:
	cd server && go run -ldflags='$(LDFLAGS)' cmd/server/main.go

dev-web:
	cd web && bun dev

swagger:
	cd server && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

gen-api: swagger
	cd web && bun run gen:api

build: build-server build-web

build-server:
	cd server && CGO_ENABLED=1 go build -ldflags='$(LDFLAGS)' -o bin/bonds-server cmd/server/main.go

build-web:
	cd web && bun run build

build-all: gen-api build-web
	rm -rf server/internal/frontend/dist/*
	cp -r web/dist/* server/internal/frontend/dist/
	cd server && CGO_ENABLED=1 go build -trimpath -ldflags="-s -w $(LDFLAGS)" -o bin/bonds-server cmd/server/main.go

test: test-server test-web

test-server:
	cd server && go test ./... -v -count=1

test-web:
	cd web && bun run test

test-e2e:
	rm -f server/bonds.db server/bonds.db-shm server/bonds.db-wal
	rm -rf server/data server/uploads
	cd web && bunx playwright test

lint:
	cd server && go vet ./...
	cd web && bun run lint

clean:
	rm -rf server/bin server/bonds.db
	rm -rf web/dist
	rm -rf server/docs
	rm -rf web/src/api/generated
	find server/internal/frontend/dist -type f ! -name '.gitkeep' -delete 2>/dev/null || true

setup:
	cd server && GOPROXY=https://goproxy.cn,direct go mod download
	cd web && bun install
