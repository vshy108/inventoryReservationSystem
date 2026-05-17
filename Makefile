.PHONY: all vet test race cover cover-html demo contention-demo server http-smoke tidy clean

all: vet test race

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./internal/...
	go tool cover -func=coverage.out | tail -1

cover-html: cover
	go tool cover -html=coverage.out

demo:
	go run ./cmd/app

contention-demo:
	go run ./cmd/contention-demo

server:
	go run ./cmd/server

http-smoke:
	bash scripts/http_smoke.sh

load-test:
	bash scripts/k6_load_test.sh

tidy:
	go mod tidy

clean:
	rm -rf bin
