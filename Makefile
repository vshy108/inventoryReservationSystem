.PHONY: all vet test race cover cover-html demo server tidy clean

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

server:
	go run ./cmd/server

tidy:
	go mod tidy

clean:
	rm -rf bin
