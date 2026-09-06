// Package cronsetup installs/removes a macOS LaunchAgent that runs
// `focuson sync` once a day — the "don't lose data" safety net. Real cron
// isn't reliable on modern macOS (it's not guaranteed to fire if the machine
// was asleep, and newer macOS versions gate it behind Full Disk Access);
// launchd is what the OS actually uses for anything scheduled, so that's
// what this drives even though the user-facing framing is "a daily cron
// job".
package cronsetup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const label = "com.focuson.dailysync"

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func logPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Logs", "focuson-sync.log"), nil
}

// resolveStableBinaryPath finds the currently-running binary's real,
// symlink-resolved path, and refuses anything that looks like a `go run`
// temp build — a LaunchAgent that points at a build-cache path that gets
// swept the moment the temp dir is cleaned would silently stop working
// forever, which is exactly the kind of failure this command exists to
// prevent.
func resolveStableBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding current executable: %w", err)
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", exe, err)
	}
	if isTempBuildPath(real) {
		return "", fmt.Errorf(
			"refusing to install: %s looks like a temporary `go run`/`go test` build, not a stable binary.\n"+
				"Build one first — e.g. `go build -o ~/bin/focuson ./cli` — then run `cron install` from that binary",
			real)
	}
	return real, nil
}

func isTempBuildPath(path string) bool {
	return strings.Contains(path, "go-build") || strings.HasPrefix(path, os.TempDir())
}

func plistContents(binaryPath, logFile string, hour, minute int) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>sync</string>
	</array>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Hour</key>
		<integer>%d</integer>
		<key>Minute</key>
		<integer>%d</integer>
	</dict>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
	<key>RunAtLoad</key>
	<false/>
</dict>
</plist>
`, label, binaryPath, hour, minute, logFile, logFile)
}

// Install writes and loads a LaunchAgent that runs `focuson sync` daily at
// hour:minute (local time, 24h). Safe to call again to change the time —
// unloads any existing job for this label first.
func Install(hour, minute int) (plistFile string, err error) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "", fmt.Errorf("invalid time %02d:%02d", hour, minute)
	}
	binaryPath, err := resolveStableBinaryPath()
	if err != nil {
		return "", err
	}
	path, err := plistPath()
	if err != nil {
		return "", err
	}
	logFile, err := logPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	// Idempotent: unload whatever's there under this label before writing
	// the new plist, so re-running install (e.g. with a different --time)
	// doesn't fight with an already-loaded job. Ignore the error — it's
	// expected to fail with "no such job" on a fresh install.
	exec.Command("launchctl", "unload", path).Run()

	if err := os.WriteFile(path, []byte(plistContents(binaryPath, logFile, hour, minute)), 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	load := exec.Command("launchctl", "load", "-w", path)
	if out, err := load.CombinedOutput(); err != nil {
		return "", fmt.Errorf("launchctl load: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return path, nil
}

// Uninstall unloads and removes the LaunchAgent, if one exists.
func Uninstall() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return nil
	}
	exec.Command("launchctl", "unload", path).Run() // best-effort; job may already be gone
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	return nil
}

// ParseTime parses "HH:MM" into (hour, minute).
func ParseTime(s string) (hour, minute int, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected HH:MM, got %q", s)
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour in %q: %w", s, err)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute in %q: %w", s, err)
	}
	return hour, minute, nil
}
