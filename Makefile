APP_NAME := file-converter
BUILD_DIR := ./bin
ENV_FILE ?= .env
LOAD_ENV = if [ -f "$(ENV_FILE)" ]; then set -a; . "$(ENV_FILE)"; set +a; fi;

.PHONY: dev build test migrate-up migrate-down smoke clean

dev:
	@$(LOAD_ENV) env=$${APP_ENV:-development}; \
	if [ "$$env" = "development" ]; then \
		go run ./cmd/migrate up || exit $$?; \
	fi; \
	go run ./cmd/web

build:
	mkdir -p $(BUILD_DIR)
	@$(LOAD_ENV) go build -o $(BUILD_DIR)/file-converter ./cmd/web
	@$(LOAD_ENV) go build -o $(BUILD_DIR)/file-converter-migrate ./cmd/migrate
	@$(LOAD_ENV) go build -o $(BUILD_DIR)/file-converter-smoke ./cmd/smoke

test:
	@$(LOAD_ENV) go test ./...

migrate-up:
	@$(LOAD_ENV) go run ./cmd/migrate up

migrate-down:
	@$(LOAD_ENV) go run ./cmd/migrate down

smoke:
	@$(LOAD_ENV) go run ./cmd/smoke

clean:
	rm -rf $(BUILD_DIR)
