// Package daemon implements a lightweight watchdog that monitors the clawfirm
// desktop app and restarts it automatically after a crash.
package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// backoffWindow is the time window for counting crashes.
	backoffWindow = 60 * time.Second
	// backoffThreshold triggers a cooldown pause.
	backoffThreshold = 3
	// backoffPause is how long to wait when backoff triggers.
	backoffPause = 30 * time.Second
	// giveUpThreshold stops restarting entirely.
	giveUpThreshold = 5
)

// DataDir returns ~/.clawfirm, creating it if needed.
func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawfirm")
}

// Paths within the data directory.
func watchdogPIDPath() string { return filepath.Join(DataDir(), "watchdog.pid") }
func appPIDPath() string      { return filepath.Join(DataDir(), "app.pid") }
func cleanExitPath() string   { return filepath.Join(DataDir(), "clean_exit") }
func logPath() string         { return filepath.Join(DataDir(), "watchdog.log") }

// Daemon is the watchdog process that keeps the clawfirm app alive.
type Daemon struct {
	AppPath string // path to clawfirm.app or binary

	mu       sync.Mutex
	stopped  bool
	crashes  []time.Time
	logger   *log.Logger
	logFile  *os.File
	cmd      *exec.Cmd
}

// Start runs the watchdog loop. It blocks until stopped via SIGTERM or Stop().
func (d *Daemon) Start() error {
	if err := os.MkdirAll(DataDir(), 0755); err != nil {
		return err
	}

	// Set up logging.
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	d.logFile = f
	d.logger = log.New(f, "", log.LstdFlags)

	// Write our own PID file.
	if err := os.WriteFile(watchdogPIDPath(), []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("write watchdog pid: %w", err)
	}

	// Handle SIGTERM/SIGINT for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		d.logger.Println("received signal, stopping watchdog")
		d.Stop()
	}()

	d.logger.Printf("watchdog started (pid %d), app=%s", os.Getpid(), d.AppPath)

	// Main restart loop.
	for {
		d.mu.Lock()
		if d.stopped {
			d.mu.Unlock()
			break
		}
		d.mu.Unlock()

		// Remove stale clean_exit marker before launching.
		os.Remove(cleanExitPath())

		exitCode := d.launchAndWait()

		d.mu.Lock()
		if d.stopped {
			d.mu.Unlock()
			d.logger.Println("watchdog stopped, not restarting")
			break
		}
		d.mu.Unlock()

		// Check for clean exit marker.
		if _, err := os.Stat(cleanExitPath()); err == nil {
			d.logger.Println("app exited cleanly (clean_exit marker found), not restarting")
			break
		}

		d.logger.Printf("app exited (code %d), checking backoff", exitCode)

		// Record crash and check backoff.
		now := time.Now()
		d.mu.Lock()
		d.crashes = append(d.crashes, now)
		// Trim old crashes outside the window.
		cutoff := now.Add(-backoffWindow)
		trimmed := d.crashes[:0]
		for _, t := range d.crashes {
			if t.After(cutoff) {
				trimmed = append(trimmed, t)
			}
		}
		d.crashes = trimmed
		recentCount := len(d.crashes)
		d.mu.Unlock()

		if recentCount >= giveUpThreshold {
			d.logger.Printf("too many crashes (%d in %v), giving up", recentCount, backoffWindow)
			break
		}

		if recentCount >= backoffThreshold {
			d.logger.Printf("%d crashes in %v, waiting %v before restart", recentCount, backoffWindow, backoffPause)
			if d.sleepOrStop(backoffPause) {
				break
			}
		}

		d.logger.Println("restarting clawfirm...")
	}

	d.cleanup()
	return nil
}

// Stop signals the watchdog to stop restarting.
func (d *Daemon) Stop() {
	d.mu.Lock()
	d.stopped = true
	d.mu.Unlock()
}

// launchAndWait starts the app and waits for it to exit, returning the exit code.
func (d *Daemon) launchAndWait() int {
	// Use "open -W -a" on macOS to launch the .app bundle and wait.
	var cmd *exec.Cmd
	if strings.HasSuffix(d.AppPath, ".app") {
		cmd = exec.Command("open", "-W", "-a", d.AppPath)
	} else {
		cmd = exec.Command(d.AppPath)
	}

	d.mu.Lock()
	d.cmd = cmd
	d.mu.Unlock()

	if err := cmd.Start(); err != nil {
		d.logger.Printf("failed to start app: %v", err)
		return 1
	}

	d.logger.Printf("app launched (pid %d)", cmd.Process.Pid)

	err := cmd.Wait()

	d.mu.Lock()
	d.cmd = nil
	d.mu.Unlock()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

// sleepOrStop waits for the given duration, returning true if stopped during the wait.
func (d *Daemon) sleepOrStop(dur time.Duration) bool {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(dur)
	for range ticker.C {
		d.mu.Lock()
		stopped := d.stopped
		d.mu.Unlock()
		if stopped {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
	}
	return false
}

func (d *Daemon) cleanup() {
	os.Remove(watchdogPIDPath())
	if d.logFile != nil {
		d.logFile.Close()
	}
}

// Status represents the current state of the daemon and app.
type Status struct {
	WatchdogRunning bool
	WatchdogPID     int
	AppRunning      bool
	AppPID          int
}

// GetStatus reads PID files to determine daemon and app status.
func GetStatus() Status {
	var s Status
	s.WatchdogPID, s.WatchdogRunning = readPIDFile(watchdogPIDPath())
	s.AppPID, s.AppRunning = readPIDFile(appPIDPath())
	return s
}

// StopDaemon sends SIGTERM to the running watchdog process.
func StopDaemon() error {
	pid, running := readPIDFile(watchdogPIDPath())
	if !running {
		return fmt.Errorf("watchdog is not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	return proc.Signal(syscall.SIGTERM)
}

// readPIDFile reads a PID from a file and checks if the process is alive.
func readPIDFile(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	// Check if the process is alive by sending signal 0.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pid, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return pid, false
	}
	return pid, true
}
