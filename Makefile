.PHONY: version-sync check-version check-release-tag build build-daemon build-daemon-debug build-steno \
       sign-daemon sign-daemon-debug \
       run-daemon run-steno run-mcp \
       build-app run-app test-app clean-app \
       test test-daemon test-steno \
       clean install

# Directories
DAEMON_DIR    = daemon
STENO_DIR     = cmd/steno
DAEMON_RELEASE = $(DAEMON_DIR)/.build/release
DAEMON_DEBUG   = $(DAEMON_DIR)/.build/debug

# Binaries
DAEMON_BIN = steno-daemon
STENO_BIN  = steno

# Signing — ad-hoc is correct for local CLI use. Apple Development
# certificates trigger provisioning profile validation which fails
# for bare CLI binaries (no bundle to embed a profile in).
CODESIGN_IDENTITY ?= -
ENTITLEMENTS      = $(DAEMON_DIR)/Resources/StenoDaemon.entitlements
INFO_PLIST        = Resources/Info.plist

# Install location — ~/.local/bin by default (no sudo needed).
# Override with: make install PREFIX=/usr/local/bin
PREFIX = $(HOME)/.local/bin

# --- Build ---

build: check-version build-daemon build-steno

build-daemon:
	cd $(DAEMON_DIR) && swift build -c release \
		-Xlinker -sectcreate -Xlinker __TEXT \
		-Xlinker __info_plist -Xlinker $(INFO_PLIST)

build-daemon-debug:
	cd $(DAEMON_DIR) && swift build \
		-Xlinker -sectcreate -Xlinker __TEXT \
		-Xlinker __info_plist -Xlinker $(INFO_PLIST)

build-steno:
	cd $(STENO_DIR) && go build -o $(STENO_BIN) .

# --- Sign ---

sign-daemon: build-daemon
	codesign --force --sign "$(CODESIGN_IDENTITY)" \
		--entitlements $(ENTITLEMENTS) \
		$(DAEMON_RELEASE)/$(DAEMON_BIN)

sign-daemon-debug: build-daemon-debug
	codesign --force --sign "$(CODESIGN_IDENTITY)" \
		--entitlements $(ENTITLEMENTS) \
		$(DAEMON_DEBUG)/$(DAEMON_BIN)

# --- Run ---

run-daemon: sign-daemon-debug
	$(DAEMON_DEBUG)/$(DAEMON_BIN) run

run-steno: build-steno
	$(STENO_DIR)/$(STENO_BIN)

run-mcp: build-steno
	$(STENO_DIR)/$(STENO_BIN) --mcp

# --- Test ---

test: check-version test-daemon test-steno test-app

# --- Version ---

version-sync:
	./scripts/sync-version.sh

# Fails if the committed version constants have drifted from VERSION.
# They are generated and committed (rather than computed at build time)
# because the Homebrew formula builds from a source tarball and calls
# `swift build` / `go build` directly, so nothing the Makefile computes
# would reach it. Committing them is what keeps all three build paths
# honest — this check is what keeps the committed copies honest.
check-version:
	@./scripts/sync-version.sh --check

# Assert VERSION matches the release tag being built. `check-version` only
# proves the generated constants agree with VERSION — it cannot notice that
# VERSION itself was never bumped, so a `v0.6.0` tag would happily publish
# binaries reporting 0.5.1. The release workflow passes the tag in as TAG.
check-release-tag:
	@if [ -z "$(TAG)" ]; then echo "check-release-tag: TAG is required (e.g. make check-release-tag TAG=v0.6.0)"; exit 1; fi
	@expected="$$(tr -d '[:space:]' < VERSION)"; \
	actual="$$(printf '%s' "$(TAG)" | sed 's/^v//')"; \
	if [ "$$expected" != "$$actual" ]; then \
		echo "ERROR: tag $(TAG) does not match VERSION ($$expected)."; \
		echo "Bump VERSION to $$actual, run \`make version-sync\`, commit, and re-tag."; \
		exit 1; \
	fi; \
	echo "Tag $(TAG) matches VERSION ($$expected)"

test-daemon:
	# Xcode 27 emits different test symbols; rely on the actual exit status.
	cd $(DAEMON_DIR) && swift test

test-steno:
	# `-p 1` serializes packages. `internal/app` and `internal/daemon` both
	# drive the single shared steno-daemon over its socket, so running them
	# concurrently means one package's recording state decides whether the
	# other's tests pass (#87).
	cd $(STENO_DIR) && go test -p 1 ./...

# --- Clean ---

clean:
	cd $(DAEMON_DIR) && swift package clean
	rm -f $(STENO_DIR)/$(STENO_BIN)

# --- Install ---

install: sign-daemon build-steno
	install -d $(PREFIX)
	install -m 755 $(DAEMON_RELEASE)/$(DAEMON_BIN) $(PREFIX)/$(DAEMON_BIN)
	install -m 755 $(STENO_DIR)/$(STENO_BIN) $(PREFIX)/$(STENO_BIN)
	@if [ -d "$(DAEMON_RELEASE)/FluidAudio_FluidAudio.bundle" ]; then \
		ditto "$(DAEMON_RELEASE)/FluidAudio_FluidAudio.bundle" "$(PREFIX)/FluidAudio_FluidAudio.bundle"; \
	fi
	@# Remove old binaries from previous three-binary layout
	@rm -f $(PREFIX)/steno-tui $(PREFIX)/steno-mcp 2>/dev/null || true
	@echo ""
	@echo "Installed to $(PREFIX):"
	@echo "  $(PREFIX)/$(DAEMON_BIN)"
	@echo "  $(PREFIX)/$(STENO_BIN)  (TUI default, --mcp for MCP server)"

# --- macOS App (SwiftUI) ---
# A native front-end to the daemon. Assembled into a signed .app bundle so it
# runs as a proper GUI/menu-bar app (the bare SwiftPM binary lacks an
# Info.plist and won't activate correctly).
APP_DIR        = app
APP_RELEASE    = $(APP_DIR)/.build/release
APP_BIN        = StenoApp
APP_BUNDLE     = $(APP_DIR)/.build/Steno.app
APP_INFO_PLIST = $(APP_DIR)/Resources/Info.plist

build-app:
	cd $(APP_DIR) && swift build -c release --product StenoApp
	@rm -rf "$(APP_BUNDLE)"
	@mkdir -p "$(APP_BUNDLE)/Contents/MacOS" "$(APP_BUNDLE)/Contents/Resources"
	@cp "$(APP_RELEASE)/$(APP_BIN)" "$(APP_BUNDLE)/Contents/MacOS/$(APP_BIN)"
	@cp "$(APP_INFO_PLIST)" "$(APP_BUNDLE)/Contents/Info.plist"
	@codesign --force --deep --sign "$(CODESIGN_IDENTITY)" "$(APP_BUNDLE)" >/dev/null 2>&1 || true
	@echo "Built $(APP_BUNDLE)"

run-app: build-app
	open "$(APP_BUNDLE)"

test-app:
	cd $(APP_DIR) && swift test

clean-app:
	cd $(APP_DIR) && swift package clean
	rm -rf "$(APP_BUNDLE)"
