.PHONY: build run dev test vet fmt swagger check cloudflare-check

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

dev:
	air -c .air.toml

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find cmd internal -name '*.go')

swagger:
	swag init -g main.go -d cmd/api,internal/app,internal/domains/fasting -o internal/domains/swagger/generated

check: test vet
	go build -o bin/api ./cmd/api

cloudflare-check:
	npm run cf:check
