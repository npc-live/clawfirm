BIN_DIR  := bin
GO       := go
CMDS     := func gateway wschat
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: all clean $(CMDS) desktop frontend app install

all: $(CMDS) desktop

# --- Pure Go binaries ---
$(CMDS):
	$(GO) build -o $(BIN_DIR)/$@ ./cmd/$@

# --- Desktop (Wails) ---
frontend:
	cd cmd/desktop/frontend && npm install && npm run build

desktop:
	cd cmd/desktop && wails build -o ../../$(BIN_DIR)/desktop

# --- macOS .app bundle ---
# Produces: bin/pi-go.app  (drag to /Applications to install)
# Embeds a darwin/universal `func` binary into app/assets/func before compiling.
app: func-universal
	cd cmd/desktop && wails build \
		-ldflags "-X github.com/ai-gateway/pi-go/app.Version=$(VERSION)" \
		-platform darwin/universal \
		-o pi-go
	@mkdir -p $(BIN_DIR)
	@rm -rf $(BIN_DIR)/pi-go.app
	@mv cmd/desktop/build/bin/pi-go.app $(BIN_DIR)/pi-go.app
	@echo ""
	@echo "✓ Built: $(BIN_DIR)/pi-go.app  ($(VERSION))"
	@echo "  Install: cp -r $(BIN_DIR)/pi-go.app /Applications/"

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

# --- Quick install to /Applications ---
install: app
	@echo "Installing pi-go.app to /Applications…"
	@rm -rf /Applications/pi-go.app
	@cp -r $(BIN_DIR)/pi-go.app /Applications/pi-go.app
	@echo "✓ Installed: /Applications/pi-go.app"

# --- Helpers ---
clean:
	rm -rf $(BIN_DIR)

test:
	$(GO) test ./cmd/func/ ./funcs/... ./skill/... ./agent/...
