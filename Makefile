.PHONY: build test race bench stress lint run clean tidy install bridge release

BINARY   = hermes-go
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -ldflags="-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/agent

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test -bench=. -benchmem -count=3 ./...

stress:
	go test -race -tags=stress -timeout 10m ./test/stress/...

lint:
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed: go install honnef.co/go/tools/cmd/staticcheck@latest"

# Instala el binario y el bridge en ~/.hermes-go/ (desarrollo local)
install: build bridge
	mkdir -p "$(HOME)/.local/bin"
	cp $(BINARY) "$(HOME)/.local/bin/$(BINARY)"
	@echo "Instalado en $(HOME)/.local/bin/$(BINARY)"

# Instala dependencias Node.js del bridge
bridge:
	cd bridge && npm install --omit=dev

run: build
	./$(BINARY) --config config.yaml

tidy:
	go mod tidy

# Compila para todos los targets de release
release:
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/hermes-go_linux_amd64   ./cmd/agent
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/hermes-go_linux_arm64   ./cmd/agent
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/hermes-go_darwin_amd64  ./cmd/agent
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/hermes-go_darwin_arm64  ./cmd/agent
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/hermes-go_windows_amd64.exe ./cmd/agent
	cd dist && for f in hermes-go_linux_* hermes-go_darwin_*; do tar -czf $${f}.tar.gz $$f; done
	cd dist && zip hermes-go_windows_amd64.zip hermes-go_windows_amd64.exe
	tar -czf dist/bridge.tar.gz bridge/
	cd dist && zip -r bridge.zip ../bridge/
	@echo "Binarios en dist/"

clean:
	rm -f $(BINARY)
	rm -rf dist/
	go clean -cache -testcache
