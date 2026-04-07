BIN_DIR  := bin
GO       := go
CMDS     := clawfirm func gateway wschat browser-shortcut
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: all clean $(CMDS) desktop frontend app install claw mobile-android mobile-ios mobile-dev-android mobile-dev-ios

all: claw $(CMDS) desktop

# --- Pure Go binaries ---
$(CMDS):
	$(GO) build -o $(BIN_DIR)/$@ ./cmd/$@

# --- Desktop (Wails) ---
frontend:
	cd cmd/desktop/frontend && npm install && npm run build

desktop:
	@rm -rf cmd/desktop/frontend/wailsjs
	cd cmd/desktop && wails build -o ../../$(BIN_DIR)/desktop

# --- claw-code Rust CLI ---
claw:
	cd claw-code/rust && cargo build --release
	@echo "✓ Built: claw-code/rust/target/release/claw"

claw-debug:
	cd claw-code/rust && cargo build
	@echo "✓ Built: claw-code/rust/target/debug/claw"

# --- macOS .app bundle ---
# Produces: bin/clawfirm.app  (drag to /Applications to install)
# Embeds a darwin/universal `func` binary and claw binary into app/assets before compiling.
app: func-universal claw browser-shortcut-universal
	@cp claw-code/rust/target/release/claw app/assets/claw
	@echo "✓ Embedded claw binary: app/assets/claw ($$(wc -c < app/assets/claw | tr -d ' ') bytes)"
	@rm -rf cmd/desktop/frontend/wailsjs
	cd cmd/desktop && wails build \
		-ldflags "-X github.com/ai-gateway/clawfirm/app.Version=$(VERSION)" \
		-platform darwin/universal \
		-o clawfirm
	@mkdir -p $(BIN_DIR)
	@rm -rf $(BIN_DIR)/clawfirm.app
	@mv cmd/desktop/build/bin/clawfirm.app $(BIN_DIR)/clawfirm.app
	@echo ""
	@echo "✓ Built: $(BIN_DIR)/clawfirm.app  ($(VERSION))"
	@echo "  Install: cp -r $(BIN_DIR)/clawfirm.app /Applications/"

# Build func as a darwin/universal binary and embed it into the app package.
func-universal:
	@echo "Building func (arm64)…"
	@GOOS=darwin GOARCH=arm64 $(GO) build -o $(BIN_DIR)/func-arm64 ./cmd/func
	@echo "Building func (amd64)…"
	@GOOS=darwin GOARCH=amd64 $(GO) build -o $(BIN_DIR)/func-amd64 ./cmd/func
	@echo "Creating universal binary…"
	@lipo -create -output app/assets/func $(BIN_DIR)/func-arm64 $(BIN_DIR)/func-amd64
	@rm -f $(BIN_DIR)/func-arm64 $(BIN_DIR)/func-amd64
	@echo "✓ Embedded func binary: app/assets/func ($$(wc -c < app/assets/func | tr -d ' ') bytes)"

# Build browser-shortcut as a darwin/universal binary and embed it into the app package.
browser-shortcut-universal:
	@echo "Building browser-shortcut (arm64)…"
	@GOOS=darwin GOARCH=arm64 $(GO) build -o $(BIN_DIR)/browser-shortcut-arm64 ./cmd/browser-shortcut
	@echo "Building browser-shortcut (amd64)…"
	@GOOS=darwin GOARCH=amd64 $(GO) build -o $(BIN_DIR)/browser-shortcut-amd64 ./cmd/browser-shortcut
	@echo "Creating universal binary…"
	@lipo -create -output app/assets/browser-shortcut $(BIN_DIR)/browser-shortcut-arm64 $(BIN_DIR)/browser-shortcut-amd64
	@rm -f $(BIN_DIR)/browser-shortcut-arm64 $(BIN_DIR)/browser-shortcut-amd64
	@echo "✓ Embedded browser-shortcut binary: app/assets/browser-shortcut ($$(wc -c < app/assets/browser-shortcut | tr -d ' ') bytes)"

# --- Quick install to /Applications ---
install: app
	@echo "Installing clawfirm.app to /Applications…"
	@rm -rf /Applications/clawfirm.app
	@cp -r $(BIN_DIR)/clawfirm.app /Applications/clawfirm.app
	@echo "✓ Installed: /Applications/clawfirm.app"

# --- Mobile (Tauri) ---
MOBILE_DIR    := mobile
TAURI         := npx tauri
MOBILE_ENV    := JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home \
                 ANDROID_HOME=$(HOME)/Library/Android/sdk \
                 NDK_HOME=$(HOME)/Library/Android/sdk/ndk/27.0.12077973

# Build debug APK (auto-signed, installable)
mobile-android:
	cd $(MOBILE_DIR) && $(MOBILE_ENV) $(TAURI) android build --apk --debug
	@echo ""
	@echo "✓ APK: $(MOBILE_DIR)/src-tauri/gen/android/app/build/outputs/apk/universal/debug/app-universal-debug.apk"
	@echo "  Install: make mobile-install-android"

# Install APK to connected device
mobile-install-android:
	$(HOME)/Library/Android/sdk/platform-tools/adb install -r \
		$(MOBILE_DIR)/src-tauri/gen/android/app/build/outputs/apk/universal/debug/app-universal-debug.apk

# Dev mode on Android device
mobile-dev-android:
	cd $(MOBILE_DIR) && $(MOBILE_ENV) $(TAURI) android dev

# Build iOS (device/simulator)
mobile-ios:
	cd $(MOBILE_DIR) && $(TAURI) ios build

# Dev mode on iOS simulator
mobile-dev-ios:
	cd $(MOBILE_DIR) && $(TAURI) ios dev

# --- Helpers ---
clean:
	rm -rf $(BIN_DIR)

test:
	$(GO) test ./cmd/func/ ./funcs/... ./skill/... ./agent/...
