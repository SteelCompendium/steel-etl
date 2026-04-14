binary := "steel-etl"
pkg    := "./cmd/steel-etl"
module := "github.com/SteelCompendium/steel-etl"

# List available recipes
default:
    @just --list

# Build the binary
build:
    go build -o {{binary}} {{pkg}}

# Run tests with race detection
test:
    go test -race ./... -v

# Run tests and show coverage summary
cover:
    go test -race ./... -coverprofile=coverage.out
    go tool cover -func=coverage.out

# Open coverage report in browser
cover-html: cover
    go tool cover -html=coverage.out

# Run go vet
vet:
    go vet ./...

# Format all Go files
fmt:
    gofmt -w .
    goimports -w . 2>/dev/null || true

# Remove build artifacts
clean:
    rm -f {{binary}} coverage.out

# Install the binary to $GOPATH/bin
install:
    go install {{pkg}}

# Run steel-etl with provided arguments
run *args:
    go run {{pkg}} {{args}}
