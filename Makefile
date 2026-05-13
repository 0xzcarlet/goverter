APP_NAME := file-converter
BUILD_DIR := ./bin

.PHONY: dev build test migrate-up migrate-down smoke clean

dev:
	go run ./cmd/web

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/file-converter ./cmd/web
	go build -o $(BUILD_DIR)/file-converter-migrate ./cmd/migrate
	go build -o $(BUILD_DIR)/file-converter-smoke ./cmd/smoke

test:
	go test ./...

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

smoke:
	go run ./cmd/smoke

clean:
	rm -rf $(BUILD_DIR)
