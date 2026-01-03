.PHONY: build build-windows clean deps fmt vet

# Build variables
BINARY_NAME=quadmax-wifi-connector.exe
GOOS=windows
GOARCH=amd64

# Build the Windows executable (requires mingw-w64 for cross-compilation from Linux)
# On Linux: sudo apt-get install gcc-mingw-w64-x86-64
build: deps
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags "-H windowsgui -s -w" -o $(BINARY_NAME) .

# Build on Windows directly (run this from a Windows machine)
build-windows: deps
	go build -ldflags "-H windowsgui -s -w" -o $(BINARY_NAME) .

# Download dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Run locally (for development on Windows)
run:
	go run .

# Format code
fmt:
	go fmt ./...

# Vet code (Windows only due to platform-specific dependencies)
vet:
	go vet ./...

# Check syntax without building (works on any platform)
check:
	go build -o /dev/null ./config/...

# Install dependencies for cross-compilation from Linux
install-cross-deps:
	@echo "Installing mingw-w64 for cross-compilation..."
	sudo apt-get update && sudo apt-get install -y gcc-mingw-w64-x86-64
