// Package gitsync commits the data directory to git — the durability layer
// spec_v2.md builds on top of plain CSV/TOML files. Never automatic (see
// spec_v2.md, Non-goals): only ever runs when the TUI's Sync screen or the
// `focuson sync` flag is explicitly invoked (including from the daily cron
// job — see internal/cronsetup — which is still an explicit `focuson sync`
// call, just on a timer instead of a keypress).
package gitsync

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ensureRepo(dataDir string) error {
	if _, err := os.Stat(filepath.Join(dataDir, ".git")); err == nil {
		return nil
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dataDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// Status returns one line per changed/untracked file (git's --porcelain
// format), or nil if the working tree is clean. Also initializes the data
// directory as a git repo on first use, if it isn't one already.
func Status(dataDir string) ([]string, error) {
	if err := ensureRepo(dataDir); err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dataDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func hasOrigin(dataDir string) bool {
	cmd := exec.Command("git", "remote")
	cmd.Dir = dataDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, name := range strings.Fields(string(out)) {
		if name == "origin" {
			return true
		}
	}
	return false
}

func currentBranch(dataDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dataDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w: %s", err, bytes.TrimSpace(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// Result reports what Commit actually did, since either half can be a no-op
// without that being a failure.
type Result struct {
	Committed bool
	Pushed    bool
	PushError error // set when an "origin" remote exists but push failed — never fatal, see Commit
}

// Commit stages and commits everything (message blank => a timestamped
// default), then — if a remote named "origin" exists — pushes, regardless of
// whether this call itself created a new commit: a previous run's commit
// could already be sitting locally, unpushed (e.g. the remote was added
// after the fact, or a prior push failed). Push failures (no network,
// diverged history, no remote configured yet, ...) are reported on the
// result's PushError but never returned as the call's error — a successful
// local commit is real durability on its own, and a flaky network shouldn't
// make the daily cron job treat every run as a failure.
func Commit(dataDir, message string) (Result, error) {
	if err := ensureRepo(dataDir); err != nil {
		return Result{}, err
	}

	lines, err := Status(dataDir)
	if err != nil {
		return Result{}, err
	}

	var result Result
	if len(lines) > 0 {
		if message == "" {
			message = fmt.Sprintf("focuson sync: %s (%d file(s) changed)", time.Now().Format("2006-01-02 15:04"), len(lines))
		}
		add := exec.Command("git", "add", "-A")
		add.Dir = dataDir
		if out, err := add.CombinedOutput(); err != nil {
			return Result{}, fmt.Errorf("git add: %w: %s", err, bytes.TrimSpace(out))
		}
		commit := exec.Command("git", "commit", "-m", message)
		commit.Dir = dataDir
		if out, err := commit.CombinedOutput(); err != nil {
			return Result{}, fmt.Errorf("git commit: %w: %s", err, bytes.TrimSpace(out))
		}
		result.Committed = true
	}

	if hasOrigin(dataDir) {
		branch, err := currentBranch(dataDir)
		if err != nil {
			result.PushError = err
			return result, nil
		}
		push := exec.Command("git", "push", "origin", branch)
		push.Dir = dataDir
		if out, err := push.CombinedOutput(); err != nil {
			result.PushError = fmt.Errorf("git push: %w: %s", err, bytes.TrimSpace(out))
		} else {
			result.Pushed = true
		}
	}

	return result, nil
}
