.PHONY: build build-ui run test test-ui test-package-verifier test-release-version fmt check-format vet package package-host package-file verify-package verify-package-host e2e e2e-live clean

# Keep these values synchronized with manifest.yaml.
BIN := bin/kandev-plugin-bitbucket
VERSION := 0.3.1
STAGE := .build/stage
PKG_OUT := $(notdir $(BIN))-$(VERSION).tar.gz
KANDEV_SDK := ../kandev/apps/backend

## Build the plugin binary for the host platform (development use). kandev
## itself always installs from `make package`/`package-host` output, not this.
build: build-ui
	mkdir -p bin
	go build -o $(BIN) ./server/...

build-ui:
	npm run build:ui

## Build + run. Mainly for -race / manual smoke checks: kandev normally spawns
## this binary itself via the go-plugin handshake, so a manually-started
## process has nothing to talk to on the other end.
run: build
	./$(BIN)

test: test-ui test-package-verifier test-release-version
	go test ./...

test-ui:
	npm test

fmt:
	gofmt -l .

check-format:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }

test-package-verifier:
	sh scripts/test-verify-package.sh

test-release-version:
	sh scripts/test-verify-release-version.sh

vet:
	go vet ./...

## Cross-compile server/plugin-<goos>-<goarch>[.exe] for every platform in
## manifest.yaml's runtime.executables, stage manifest.yaml + assets/ + ui/
## alongside them, and pack the tree into $(PKG_OUT) with
## Kandev's cmd/plugin-pack using the sibling SDK module. Running the tool in
## that module keeps its tool-only dependencies owned by Kandev rather than
## leaking them into this plugin's go.sum. Install the tarball via Settings >
## Plugins or curl -F package=@...
package: build-ui
	rm -rf $(STAGE)
	mkdir -p $(STAGE)/server
	cp manifest.yaml $(STAGE)/manifest.yaml
	cp -r assets $(STAGE)/assets
	mkdir -p $(STAGE)/ui
	cp ui/bundle.js ui/plugin.css $(STAGE)/ui/
	GOOS=linux   GOARCH=amd64 go build -o $(STAGE)/server/plugin-linux-amd64       ./server
	GOOS=linux   GOARCH=arm64 go build -o $(STAGE)/server/plugin-linux-arm64       ./server
	GOOS=darwin  GOARCH=amd64 go build -o $(STAGE)/server/plugin-darwin-amd64      ./server
	GOOS=darwin  GOARCH=arm64 go build -o $(STAGE)/server/plugin-darwin-arm64      ./server
	GOOS=windows GOARCH=amd64 go build -o $(STAGE)/server/plugin-windows-amd64.exe ./server
	go -C $(KANDEV_SDK) run ./cmd/plugin-pack -dir $(abspath $(STAGE)) -out $(abspath $(PKG_OUT))
	rm -rf $(STAGE)
	@echo "Wrote $(PKG_OUT)"

## Package for the host platform only — faster local iteration than the full
## 5-platform `make package` (matches plugin-pack's -platform-only).
package-host: build-ui
	rm -rf $(STAGE)
	mkdir -p $(STAGE)/server
	cp manifest.yaml $(STAGE)/manifest.yaml
	cp -r assets $(STAGE)/assets
	mkdir -p $(STAGE)/ui
	cp ui/bundle.js ui/plugin.css $(STAGE)/ui/
	go build -o $(STAGE)/server/plugin-$$(go env GOOS)-$$(go env GOARCH)$$(go env GOEXE) ./server
	go -C $(KANDEV_SDK) run ./cmd/plugin-pack -dir $(abspath $(STAGE)) -out $(abspath $(PKG_OUT)) -platform-only
	rm -rf $(STAGE)
	@echo "Wrote $(PKG_OUT)"

## Verify the generated archive, not only the staged source tree. The host
## installer performs the same checksum check; this gate also proves every
## executable the selected package form promises is actually present.
verify-package: package
	@set -eu; \
	TMP_ROOT="$${TMPDIR:-$(CURDIR)/.build/tmp}"; \
	mkdir -p "$$TMP_ROOT"; \
	VERIFY_DIR="$$(mktemp -d "$$TMP_ROOT/bitbucket-package.XXXXXX")"; \
	trap 'rm -rf "$$VERIFY_DIR"' EXIT; \
	test -f "$(PKG_OUT)" || { echo "package not found: $(PKG_OUT)"; exit 1; }; \
	tar -xzf "$(PKG_OUT)" -C "$$VERIFY_DIR"; \
	sh scripts/verify-package.sh "$$VERIFY_DIR" full

verify-package-host: package-host
	@set -eu; \
	TMP_ROOT="$${TMPDIR:-$(CURDIR)/.build/tmp}"; \
	mkdir -p "$$TMP_ROOT"; \
	VERIFY_DIR="$$(mktemp -d "$$TMP_ROOT/bitbucket-package.XXXXXX")"; \
	trap 'rm -rf "$$VERIFY_DIR"' EXIT; \
	test -f "$(PKG_OUT)" || { echo "package not found: $(PKG_OUT)"; exit 1; }; \
	tar -xzf "$(PKG_OUT)" -C "$$VERIFY_DIR"; \
	sh scripts/verify-package.sh "$$VERIFY_DIR" host "$$(go env GOOS)-$$(go env GOARCH)"

package-file:
	@printf '%s\n' "$(PKG_OUT)"

## The contract runner uploads the freshly packaged archive to a fresh
## disposable compatible host. It fails before Playwright starts when the host
## URL is absent or the artifact cannot be installed and activated.
e2e: package-host verify-package-host
	KANDEV_PLUGIN_E2E_PACKAGE="$(abspath $(PKG_OUT))" npm run e2e:contract

## Opt-in configured-provider acceptance. Unlike the normal packaged contract,
## this requires a disposable Bitbucket target and secret environment variables
## documented in README.md. The live Playwright config disables traces,
## screenshots, and video so credentials never enter test artifacts.
e2e-live: package-host verify-package-host
	KANDEV_PLUGIN_E2E_PACKAGE="$(abspath $(PKG_OUT))" npm run e2e:live

clean:
	rm -rf bin $(STAGE) .build kandev-plugin-bitbucket-*.tar.gz
