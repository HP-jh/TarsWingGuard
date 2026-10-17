.PHONY: all build test clean install cross release

VERSION := 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/HP-jh/TarsWingGuard/internal/version.Commit=$(COMMIT) \
           -X github.com/HP-jh/TarsWingGuard/internal/version.BuildTime=$(BUILD_TIME)

BINARIES := wingctl wing-guard

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/wingctl ./cmd/wingctl
	go build -ldflags "$(LDFLAGS)" -o bin/wing-guard ./cmd/wing-guard

test:
	go test -race -count=1 ./...

clean:
	rm -rf bin/ dist/

install: build
	cp bin/wingctl $(GOPATH)/bin/twg 2>/dev/null || cp bin/wingctl /usr/local/bin/twg
	cp bin/wing-guard $(GOPATH)/bin/ 2>/dev/null || cp bin/wing-guard /usr/local/bin/

# Cross-compile for all 7 platforms + android
cross:
	mkdir -p dist
	# linux amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-linux-amd64 ./cmd/wingctl
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-linux-amd64 ./cmd/wing-guard
	# linux arm64
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-linux-arm64 ./cmd/wingctl
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-linux-arm64 ./cmd/wing-guard
	# darwin amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-darwin-amd64 ./cmd/wingctl
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-darwin-amd64 ./cmd/wing-guard
	# darwin arm64
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-darwin-arm64 ./cmd/wingctl
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-darwin-arm64 ./cmd/wing-guard
	# windows amd64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-windows-amd64.exe ./cmd/wingctl
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-windows-amd64.exe ./cmd/wing-guard
	# windows arm64
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-windows-arm64.exe ./cmd/wingctl
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-windows-arm64.exe ./cmd/wing-guard
	# android arm64 (via Go cross-compile)
	GOOS=android GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/twg-v$(VERSION)-android-arm64 ./cmd/wingctl 2>/dev/null || true
	GOOS=android GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/wing-guard-v$(VERSION)-android-arm64 ./cmd/wing-guard 2>/dev/null || true

release: cross
	cd dist && sha256sum * > SHA256SUMS.txt 2>/dev/null || shasum -a 256 * > SHA256SUMS.txt
	@echo "Release artifacts in dist/"
	@ls -la dist/
