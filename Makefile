.PHONY: help run run-supermarket run-all test fmt build check

-include .env
export POCKETBASE_URL POCKETBASE_IDENTITY POCKETBASE_PASSWORD STATS_WORKERS

help:
	@echo "make run    Ejecutar el servicio"
	@echo "make run-supermarket SUPERMARKET_ID=...  Ejecutar un supermercado"
	@echo "make run-all  Ejecutar todos los supermercados una vez"
	@echo "make test   Ejecutar los tests"
	@echo "make fmt    Formatear el código Go"
	@echo "make build  Compilar el servicio"
	@echo "make check  Formatear y ejecutar tests"

run:
	go run ./cmd/stats-service

run-supermarket:
	@test -n "$(SUPERMARKET_ID)" || (echo "Uso: make run-supermarket SUPERMARKET_ID=..."; exit 1)
	go run ./cmd/stats-service --run-supermarket "$(SUPERMARKET_ID)"

run-all:
	go run ./cmd/stats-service --run-all

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

build:
	go build -o bin/stats-service ./cmd/stats-service

check: fmt test
