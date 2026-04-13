BINARY := steel-etl
PKG    := ./cmd/steel-etl
MODULE := github.com/SteelCompendium/steel-etl

.PHONY: build test cover lint clean install

build:
	go build -o $(BINARY) $(PKG)

test:
	go test ./... -v

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	go vet ./...

clean:
	rm -f $(BINARY) coverage.out

install:
	go install $(PKG)
