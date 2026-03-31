package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ai-gateway/clawfirm/vault"
	"github.com/ai-gateway/clawfirm/vault/keychain"
)

func runVault(args []string) {
	if len(args) == 0 {
		printVaultUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "init":
		vaultInit()
	case "set":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault set <KEY>")
		}
		vaultSet(args[1])
	case "get":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault get <KEY>")
		}
		vaultGet(args[1])
	case "list", "ls":
		vaultList()
	case "delete", "rm":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault delete <KEY>")
		}
		vaultDelete(args[1])
	case "env":
		vaultEnv()
	case "run":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault run <cmd> [args...]")
		}
		vaultRun(args[1:])
	case "resolve":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault resolve <file.json>")
		}
		vaultResolve(args[1:])
	case "shell-init":
		shell := ""
		for i := 1; i < len(args); i++ {
			if (args[i] == "--shell" || args[i] == "-shell") && i+1 < len(args) {
				shell = args[i+1]
				break
			}
		}
		if shell == "" {
			log.Fatal("usage: clawfirm vault shell-init --shell <zsh|bash>")
		}
		vaultShellInit(shell)
	case "install":
		vaultInstall()
	case "uninstall":
		vaultUninstall()
	case "variant":
		if len(args) < 2 {
			printVariantUsage()
			os.Exit(1)
		}
		runVariant(args[1:])
	case "help", "-h", "--help":
		printVaultUsage()
	default:
		fmt.Fprintf(os.Stderr, "vault: unknown command %q\n\n", args[0])
		printVaultUsage()
		os.Exit(1)
	}
}

func printVaultUsage() {
	fmt.Fprintln(os.Stderr, `Usage: clawfirm vault <command> [args]

Commands:
  init                       Initialize encrypted vault
  set <KEY>                  Set secret (hidden input)
  get <KEY>                  Get secret value
  list                       List all secret names
  delete <KEY>               Delete a secret
  env                        Print export statements
  run <cmd> [args...]        Run command with vault env injected
  resolve <file.json>        Resolve _ref fields in JSON
  shell-init --shell <shell> Print shell hook code (zsh, bash)
  install                    Add vault shell hook to ~/.zshrc or ~/.bashrc
  uninstall                  Remove vault shell hook from shell rc file
  variant                    Manage secret variants
  help                       Show this help`)
}

func printVariantUsage() {
	fmt.Fprintln(os.Stderr, `Usage: clawfirm vault variant <command> [args]

Commands:
  add <KEY>                  Add a new variant (hidden input)
  use <KEY> <N>              Switch active variant
  list <KEY>                 List variants for a key
  rm <KEY> <N>               Remove a variant`)
}

func openVault() *vault.Vault {
	kc := keychain.New()
	v, err := vault.Open(vault.DefaultDBPath(), kc)
	if err != nil {
		log.Fatalf("vault: %v\nRun 'clawfirm vault init' first.", err)
	}
	return v
}

func vaultInit() {
	dbPath := vault.DefaultDBPath()
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Fprintln(os.Stderr, "Vault already exists:", dbPath)
		return
	}

	kc := keychain.New()
	if err := vault.Init(dbPath, kc); err != nil {
		log.Fatalf("vault init: %v", err)
	}
	fmt.Println("Vault initialized:", dbPath)
}

func vaultSet(key string) {
	secret, err := vault.ReadSecret("Enter value: ")
	if err != nil {
		log.Fatalf("vault set: %v", err)
	}
	if len(secret) == 0 {
		log.Fatal("vault set: empty value")
	}

	v := openVault()
	defer v.Close()

	if err := v.Set(key, secret); err != nil {
		log.Fatalf("vault set: %v", err)
	}
	fmt.Printf("Set: %s\n", key)
}

func vaultGet(key string) {
	v := openVault()
	defer v.Close()

	val, err := v.Get(key)
	if err != nil {
		log.Fatalf("vault get: %v", err)
	}
	fmt.Print(string(val))
}

func vaultList() {
	v := openVault()
	defer v.Close()

	keys, err := v.List()
	if err != nil {
		log.Fatalf("vault list: %v", err)
	}
	for _, k := range keys {
		fmt.Println(k)
	}
}

func vaultDelete(key string) {
	v := openVault()
	defer v.Close()

	if err := v.Delete(key); err != nil {
		log.Fatalf("vault delete: %v", err)
	}
	fmt.Printf("Deleted: %s\n", key)
}

func vaultEnv() {
	v := openVault()
	defer v.Close()

	env, err := v.Env()
	if err != nil {
		log.Fatalf("vault env: %v", err)
	}
	for k, val := range env {
		fmt.Printf("export %s=%s\n", k, shellQuote(val))
	}
}

func vaultRun(args []string) {
	v := openVault()

	env, err := v.Env()
	if err != nil {
		log.Fatalf("vault run: %v", err)
	}
	v.Close()

	binary, err := exec.LookPath(args[0])
	if err != nil {
		log.Fatalf("vault run: %v", err)
	}

	// Build environment: overlay vault secrets onto current env.
	environ := os.Environ()
	for k, val := range env {
		environ = append(environ, k+"="+val)
	}

	if err := syscall.Exec(binary, args, environ); err != nil {
		log.Fatalf("vault run: exec: %v", err)
	}
}

type refEntry struct {
	outputKey string
	vaultKey  string
}

func vaultResolve(args []string) {
	export := false
	filePath := ""
	for _, arg := range args {
		if arg == "--export" || arg == "-export" {
			export = true
		} else if filePath == "" {
			filePath = arg
		}
	}
	if filePath == "" {
		log.Fatal("usage: clawfirm vault resolve [--export] <file.json>")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("vault resolve: %v", err)
	}

	var input map[string]any
	if err := json.Unmarshal(data, &input); err != nil {
		log.Fatalf("vault resolve: parse JSON: %v", err)
	}

	// Collect _ref entries.
	var refs []refEntry
	for k, val := range input {
		if strings.HasSuffix(k, "_ref") {
			vaultKey, ok := val.(string)
			if !ok {
				log.Fatalf("vault resolve: %q value must be a string", k)
			}
			outputKey := strings.TrimSuffix(k, "_ref")
			refs = append(refs, refEntry{outputKey: outputKey, vaultKey: vaultKey})
		}
	}
	if len(refs) == 0 {
		log.Fatal("vault resolve: no _ref fields found in file")
	}

	v := openVault()
	defer v.Close()

	// Build output: non-ref fields + resolved refs.
	output := make(map[string]any)
	for k, val := range input {
		if !strings.HasSuffix(k, "_ref") {
			output[k] = val
		}
	}
	for _, ref := range refs {
		val, err := v.Get(ref.vaultKey)
		if err != nil {
			log.Fatalf("vault resolve: secret %q: %v", ref.vaultKey, err)
		}
		output[ref.outputKey] = string(val)
	}

	if export {
		for k, val := range output {
			if s, ok := val.(string); ok {
				fmt.Printf("export %s=%s\n", k, shellQuote(s))
			}
		}
		return
	}

	out, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("vault resolve: %v", err)
	}
	fmt.Println(string(out))
}

func vaultShellInit(shell string) {
	exe, err := os.Executable()
	if err != nil {
		exe = "clawfirm"
	}

	switch shell {
	case "zsh":
		fmt.Printf(`_clawfirm_preexec() {
  eval "$(%s vault env 2>/dev/null)"
}
autoload -Uz add-zsh-hook
add-zsh-hook preexec _clawfirm_preexec
`, exe)
	case "bash":
		fmt.Printf(`_clawfirm_inject() {
  eval "$(%s vault env 2>/dev/null)"
}
if [[ -z "$PROMPT_COMMAND" ]]; then
  PROMPT_COMMAND="_clawfirm_inject"
else
  PROMPT_COMMAND="_clawfirm_inject;$PROMPT_COMMAND"
fi
`, exe)
	default:
		log.Fatalf("vault shell-init: unsupported shell %q (use zsh or bash)", shell)
	}
}

const hookMarker = "# clawfirm vault"

// vaultInstall appends the shell hook to ~/.zshrc or ~/.bashrc so vault
// secrets are automatically injected as env vars in every new shell.
func vaultInstall() {
	shell, rcPath := detectShellRC()

	// Check if already installed.
	if hookInstalled(rcPath) {
		fmt.Printf("Already installed in %s\n", rcPath)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		exe = "clawfirm"
	}
	// Resolve to absolute path for reliability.
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}

	line := fmt.Sprintf(`%s`+"\n"+`eval "$(%s vault shell-init --shell %s)"`, hookMarker, exe, shell)

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("vault install: %v", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s\n", line); err != nil {
		log.Fatalf("vault install: write: %v", err)
	}
	fmt.Printf("Installed vault shell hook in %s\n", rcPath)
	fmt.Println("Restart your shell or run:  source", rcPath)
}

// vaultUninstall removes the shell hook from ~/.zshrc or ~/.bashrc.
func vaultUninstall() {
	_, rcPath := detectShellRC()

	data, err := os.ReadFile(rcPath)
	if err != nil {
		log.Fatalf("vault uninstall: %v", err)
	}

	var kept []string
	skip := false
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, hookMarker) {
			skip = true // skip this line and the next eval line
			continue
		}
		if skip {
			skip = false
			continue
		}
		kept = append(kept, line)
	}

	// Remove trailing empty lines left behind.
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}

	output := strings.Join(kept, "\n") + "\n"
	if err := os.WriteFile(rcPath, []byte(output), 0644); err != nil {
		log.Fatalf("vault uninstall: write: %v", err)
	}
	fmt.Printf("Removed vault shell hook from %s\n", rcPath)
}

// detectShellRC returns the shell name and rc file path.
func detectShellRC() (string, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("vault install: cannot determine home directory")
	}

	// Prefer SHELL env var, fall back to checking if .zshrc exists.
	sh := filepath.Base(os.Getenv("SHELL"))
	switch sh {
	case "zsh":
		return "zsh", filepath.Join(home, ".zshrc")
	case "bash":
		return "bash", filepath.Join(home, ".bashrc")
	default:
		// Default to zsh on macOS, bash elsewhere.
		if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
			return "zsh", filepath.Join(home, ".zshrc")
		}
		return "bash", filepath.Join(home, ".bashrc")
	}
}

// hookInstalled checks if the hook marker already exists in the rc file.
func hookInstalled(rcPath string) bool {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), hookMarker)
}

func runVariant(args []string) {
	switch args[0] {
	case "add":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault variant add <KEY>")
		}
		variantAdd(args[1])
	case "use":
		if len(args) < 3 {
			log.Fatal("usage: clawfirm vault variant use <KEY> <N>")
		}
		idx, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalf("variant use: invalid index %q", args[2])
		}
		variantUse(args[1], idx)
	case "list", "ls":
		if len(args) < 2 {
			log.Fatal("usage: clawfirm vault variant list <KEY>")
		}
		variantList(args[1])
	case "rm", "remove":
		if len(args) < 3 {
			log.Fatal("usage: clawfirm vault variant rm <KEY> <N>")
		}
		idx, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalf("variant rm: invalid index %q", args[2])
		}
		variantRemove(args[1], idx)
	default:
		fmt.Fprintf(os.Stderr, "variant: unknown command %q\n\n", args[0])
		printVariantUsage()
		os.Exit(1)
	}
}

func variantAdd(key string) {
	secret, err := vault.ReadSecret("Enter value: ")
	if err != nil {
		log.Fatalf("variant add: %v", err)
	}
	if len(secret) == 0 {
		log.Fatal("variant add: empty value")
	}

	v := openVault()
	defer v.Close()

	idx, err := v.VariantAdd(key, secret)
	if err != nil {
		log.Fatalf("variant add: %v", err)
	}
	fmt.Printf("Added variant #%d for %s\n", idx, key)
}

func variantUse(key string, idx int) {
	v := openVault()
	defer v.Close()

	if err := v.VariantUse(key, idx); err != nil {
		log.Fatalf("variant use: %v", err)
	}
	fmt.Printf("Switched %s to variant #%d\n", key, idx)
}

func variantList(key string) {
	v := openVault()
	defer v.Close()

	entries, err := v.VariantList(key)
	if err != nil {
		log.Fatalf("variant list: %v", err)
	}
	if len(entries) == 0 {
		fmt.Printf("No variants for %s\n", key)
		return
	}
	for _, e := range entries {
		marker := "  "
		if e.Active {
			marker = "* "
		}
		// Show first 8 chars of value for identification.
		preview := string(e.Value)
		if len(preview) > 8 {
			preview = preview[:8] + "..."
		}
		fmt.Printf("%s#%d  %s\n", marker, e.Index, preview)
	}
}

func variantRemove(key string, idx int) {
	v := openVault()
	defer v.Close()

	if err := v.VariantRemove(key, idx); err != nil {
		log.Fatalf("variant rm: %v", err)
	}
	fmt.Printf("Removed variant #%d from %s\n", idx, key)
}

// shellQuote wraps val in single quotes, escaping embedded single quotes.
func shellQuote(val string) string {
	return "'" + strings.ReplaceAll(val, "'", `'\''`) + "'"
}
