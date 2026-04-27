BIN_DIR  := bin
GO       := go
CMDS     := clawfirm func gateway wschat browser-shortcut media-understand media-gen whip
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go binaries that get embedded as universal (arm64+amd64) into app/assets for the .app bundle
UNIVERSAL_CMDS := func whip browser-shortcut media-understand media-gen

.PHONY: all clean $(CMDS) desktop frontend gateway-frontend app install linux linux-arm64 \
        mobile-android mobile-ios mobile-dev-android mobile-dev-ios mobile-install-android test

all: $(CMDS) desktop

# ── Pure Go binaries ──────────────────────────────────────────────
$(CMDS):
	$(GO) build -o $(BIN_DIR)/$@ ./cmd/$@

# ── Gateway frontend (build into cmd/gateway/frontend/dist for embed) ────
gateway-frontend:
	cd cmd/desktop/frontend && npm install && npm run build
	rm -rf cmd/gateway/frontend/dist
	cp -r cmd/desktop/frontend/dist cmd/gateway/frontend/dist

# ── Linux cross-compile ──────────────────────────────────────────
linux: gateway-frontend
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BIN_DIR)/linux/clawfirm ./cmd/clawfirm
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BIN_DIR)/linux/gateway ./cmd/gateway

linux-arm64: gateway-frontend
	GOOS=linux GOARCH=arm64 $(GO) build -o $(BIN_DIR)/linux-arm64/clawfirm ./cmd/clawfirm
	GOOS=linux GOARCH=arm64 $(GO) build -o $(BIN_DIR)/linux-arm64/gateway ./cmd/gateway

# ── Desktop (Wails) ──────────────────────────────────────────────
frontend:
	cd cmd/desktop/frontend && npm install && npm run build

desktop:
	@rm -rf cmd/desktop/frontend/wailsjs
	cd cmd/desktop && wails generate module
	@mkdir -p cmd/desktop/build/bin
	cd cmd/desktop && wails build -o ../../$(BIN_DIR)/desktop

# ── macOS .app bundle ────────────────────────────────────────────
# Produces: bin/clawfirm.app  (drag to /Applications to install)
# Embeds darwin/universal Go binaries + claw into app/assets before compiling.
ENTITLEMENTS := cmd/desktop/build/darwin/entitlements.plist

app: $(addsuffix -universal,$(UNIVERSAL_CMDS))
	@rm -rf cmd/desktop/frontend/wailsjs cmd/desktop/build/bin
	cd cmd/desktop && wails generate module
	cd cmd/desktop && wails build \
		-ldflags "-X github.com/ai-gateway/clawfirm/app.Version=$(VERSION)" \
		-platform darwin/universal \
		-o clawfirm
	@mkdir -p $(BIN_DIR)
	@rm -rf $(BIN_DIR)/clawfirm.app
	@mv cmd/desktop/build/bin/clawfirm.app $(BIN_DIR)/clawfirm.app
	codesign --force --deep --sign - --entitlements $(ENTITLEMENTS) $(BIN_DIR)/clawfirm.app
	@echo ""
	@echo "✓ Built: $(BIN_DIR)/clawfirm.app  ($(VERSION))"
	@echo "  Install: cp -r $(BIN_DIR)/clawfirm.app /Applications/"

# Generic rule: build any Go cmd as darwin/universal and embed into app/assets
%-universal:
	@echo "Building $* (arm64)…"
	@GOOS=darwin GOARCH=arm64 $(GO) build -o $(BIN_DIR)/$*-arm64 ./cmd/$*
	@echo "Building $* (amd64)…"
	@GOOS=darwin GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$*-amd64 ./cmd/$*
	@echo "Creating universal binary…"
	@lipo -create -output app/assets/$* $(BIN_DIR)/$*-arm64 $(BIN_DIR)/$*-amd64
	@rm -f $(BIN_DIR)/$*-arm64 $(BIN_DIR)/$*-amd64
	@echo "✓ Embedded $* binary: app/assets/$* ($$(wc -c < app/assets/$* | tr -d ' ') bytes)"

# ── Quick install to /Applications ───────────────────────────────
install: app
	@echo "Installing clawfirm.app to /Applications…"
	@rm -rf /Applications/clawfirm.app
	@cp -r $(BIN_DIR)/clawfirm.app /Applications/clawfirm.app
	@echo "✓ Installed: /Applications/clawfirm.app"

# ── Mobile (Tauri) ───────────────────────────────────────────────
MOBILE_DIR    := mobile
TAURI         := npx tauri
MOBILE_ENV    := JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home \
                 ANDROID_HOME=$(HOME)/Library/Android/sdk \
                 NDK_HOME=$(HOME)/Library/Android/sdk/ndk/27.0.12077973

mobile-android:
	cd $(MOBILE_DIR) && $(MOBILE_ENV) $(TAURI) android build --apk --debug
	@echo ""
	@echo "✓ APK: $(MOBILE_DIR)/src-tauri/gen/android/app/build/outputs/apk/universal/debug/app-universal-debug.apk"
	@echo "  Install: make mobile-install-android"

mobile-install-android:
	$(HOME)/Library/Android/sdk/platform-tools/adb install -r \
		$(MOBILE_DIR)/src-tauri/gen/android/app/build/outputs/apk/universal/debug/app-universal-debug.apk

mobile-dev-android:
	cd $(MOBILE_DIR) && $(MOBILE_ENV) $(TAURI) android dev

mobile-ios:
	cd $(MOBILE_DIR) && $(TAURI) ios build

mobile-dev-ios:
	cd $(MOBILE_DIR) && $(TAURI) ios dev

# ── Helpers ───────────────────────────────────────────────────────
clean:
	rm -rf $(BIN_DIR)
	rm -rf cmd/desktop/build/bin cmd/desktop/build/darwin
	rm -rf cmd/desktop/frontend/dist cmd/desktop/frontend/wailsjs
	rm -f app/assets/func app/assets/whip app/assets/browser-shortcut app/assets/media-understand app/assets/media-gen

test:
	$(GO) test ./cmd/func/ ./funcs/... ./skill/... ./agent/...

# ── Bug Hunter ───────────────────────────────────────────────────────────────
logcheck:
	$(GO) run ./cmd/logcheck --since 30m ~/.clawfirm/app.log

logcheck-json:
	$(GO) run ./cmd/logcheck --format json --since 30m ~/.clawfirm/app.log

scenario-runner:
	$(GO) run ./cmd/scenario-runner -v scenarios/basic_chat.yaml

bughunt:
	$(GO) run ./cmd/run-whip app/assets/workflows/dev/bughunter.whip

test-env:
	./scripts/start-test-env.sh

test-ui-basic:
	$(GO) run ./cmd/browser-shortcut ~/.clawfirm/shortcuts/clawfirm_test.yaml basic_chat '你好'

test-ui-stop:
	$(GO) run ./cmd/browser-shortcut ~/.clawfirm/shortcuts/clawfirm_test.yaml send_then_stop
