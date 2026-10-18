.PHONY: build run test clean cross-compile

BINARY=noytads
PKG=./cmd/noytads

# Development build (host architecture)
build:
	go build -o $(BINARY) $(PKG)

# Run locally for development
run: build
	./$(BINARY) -listen :8443 -ca-path ./ca.pem -ca-key ./ca.key -log-level debug

# Run tests
test:
	go test ./...

# Smoke test against running proxy
smoke-test:
	bash scripts/test-injection.sh localhost:8443 ./ca.pem

# Cross-compile for common OpenWrt targets
cross-compile:
	@echo "Building for OpenWrt targets..."
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-s -w" -o $(BINARY)-mipsle $(PKG)
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)-arm64 $(PKG)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-amd64 $(PKG)
	@ls -lh $(BINARY)-*

# Build for specific router (example: GL.iNet MT3000, aarch64)
build-router:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)-arm64 $(PKG)
	@echo "Binary ready: $(BINARY)-arm64 ($$(du -h $(BINARY)-arm64 | cut -f1))"

clean:
	rm -f $(BINARY) $(BINARY)-*
	rm -f ca.pem ca.key
