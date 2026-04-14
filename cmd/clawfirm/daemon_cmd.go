package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ai-gateway/clawfirm/daemon"
)

func runDaemon(args []string) {
	if len(args) == 0 {
		printDaemonUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "start":
		runDaemonStart(args[1:])
	case "stop":
		runDaemonStop()
	case "status":
		runDaemonStatus()
	case "help", "-h", "--help":
		printDaemonUsage()
	default:
		fmt.Fprintf(os.Stderr, "clawfirm daemon: unknown subcommand %q\n\n", args[0])
		printDaemonUsage()
		os.Exit(1)
	}
}

func printDaemonUsage() {
	fmt.Fprintln(os.Stderr, `Usage: clawfirm daemon <command>

Commands:
  start [--app <path>]   Start the watchdog daemon (backgrounds itself)
  stop                   Stop the watchdog daemon
  status                 Show watchdog and app status`)
}

func runDaemonStart(args []string) {
	// Check if already running.
	st := daemon.GetStatus()
	if st.WatchdogRunning {
		fmt.Printf("Watchdog is already running (pid %d)\n", st.WatchdogPID)
		return
	}

	// Determine app path.
	appPath := defaultAppPath()
	for i := 0; i < len(args); i++ {
		if (args[i] == "--app" || args[i] == "-app") && i+1 < len(args) {
			appPath = args[i+1]
			i++
		}
	}

	// Check if --foreground is requested (for debugging).
	foreground := false
	for _, a := range args {
		if a == "--foreground" || a == "-foreground" {
			foreground = true
		}
	}

	if foreground {
		d := &daemon.Daemon{AppPath: appPath}
		if err := d.Start(); err != nil {
			log.Fatalf("daemon: %v", err)
		}
		return
	}

	// Re-exec ourselves in the background with --foreground.
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("daemon: resolve executable: %v", err)
	}

	cmd := exec.Command(exe, "daemon", "start", "--foreground", "--app", appPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	// Detach from parent process group.
	cmd.SysProcAttr = daemonSysProcAttr()

	if err := cmd.Start(); err != nil {
		log.Fatalf("daemon: start background: %v", err)
	}

	fmt.Printf("Watchdog started (pid %d), app=%s\n", cmd.Process.Pid, appPath)

	// Detach — don't wait for the child.
	cmd.Process.Release()
}

func runDaemonStop() {
	if err := daemon.StopDaemon(); err != nil {
		log.Fatalf("daemon stop: %v", err)
	}
	fmt.Println("Watchdog stopped.")
}

func runDaemonStatus() {
	st := daemon.GetStatus()

	if st.WatchdogRunning {
		fmt.Printf("Watchdog: running (pid %d)\n", st.WatchdogPID)
	} else {
		fmt.Println("Watchdog: not running")
	}

	if st.AppRunning {
		fmt.Printf("App:      running (pid %d)\n", st.AppPID)
	} else {
		fmt.Println("App:      not running")
	}
}

func defaultAppPath() string {
	if runtime.GOOS == "darwin" {
		// Check standard macOS install locations.
		candidates := []string{
			"/Applications/clawfirm.app",
			filepath.Join(os.Getenv("HOME"), "Applications", "clawfirm.app"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
		return "/Applications/clawfirm.app"
	}
	// On Linux/other, assume the binary is in PATH.
	return "clawfirm"
}
