.PHONY: all vet test race demo server tidy clean

all: vet test race

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

demo:
	go run ./cmd/app

server:
	go run ./cmd/server

tidy:
	go mod tidy

clean:
	rm -rf bin
