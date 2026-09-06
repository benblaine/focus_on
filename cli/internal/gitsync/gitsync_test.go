package gitsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// Hermetic: don't depend on the machine's global git config having a
	// user identity set.
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func TestStatusInitializesRepoAndReportsUntracked(t *testing.T) {
	skipIfNoGit(t)
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := Status(dataDir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 untracked file, got %v", lines)
	}
	if _, err := os.Stat(filepath.Join(dataDir, ".git")); err != nil {
		t.Fatalf("expected Status to have initialized a git repo: %v", err)
	}
}

func TestCommitStagesAndCommits(t *testing.T) {
	skipIfNoGit(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644)

	result, err := Commit(dataDir, "test commit")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if !result.Committed {
		t.Fatalf("expected Committed=true, got %+v", result)
	}
	if result.Pushed || result.PushError != nil {
		t.Fatalf("expected no push attempt with no remote configured, got %+v", result)
	}

	lines, err := Status(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected clean working tree after commit, got %v", lines)
	}
}

func TestCommitOnCleanTreeWithNoRemoteIsANoop(t *testing.T) {
	skipIfNoGit(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644)

	if _, err := Commit(dataDir, "first commit"); err != nil {
		t.Fatalf("first Commit: %v", err)
	}

	result, err := Commit(dataDir, "second commit")
	if err != nil {
		t.Fatalf("second Commit: %v", err)
	}
	if result.Committed {
		t.Fatalf("expected Committed=false on a clean tree, not another commit: %+v", result)
	}
}

func TestCommitUsesDefaultMessageWhenBlank(t *testing.T) {
	skipIfNoGit(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644)

	if _, err := Commit(dataDir, ""); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = dataDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatalf("expected a non-empty default commit message")
	}
}

// addOrigin sets up a local bare repo as "origin" with tracking already
// configured, so push tests exercise the real `git push` path without
// needing network access or a real GitHub remote.
func addOrigin(t *testing.T, dataDir string) {
	t.Helper()
	bareDir := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", bareDir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	cmd := exec.Command("git", "remote", "add", "origin", bareDir)
	cmd.Dir = dataDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v: %s", err, out)
	}
}

func TestCommitPushesWhenOriginConfigured(t *testing.T) {
	skipIfNoGit(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644)
	if err := ensureRepo(dataDir); err != nil {
		t.Fatal(err)
	}
	addOrigin(t, dataDir)

	result, err := Commit(dataDir, "with a remote")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if !result.Committed || !result.Pushed || result.PushError != nil {
		t.Fatalf("expected a clean committed+pushed result, got %+v", result)
	}
}

func TestCommitPushesAlreadyCommittedWorkOnACleanTree(t *testing.T) {
	// The scenario this exists for: a remote gets added *after* a commit
	// already happened locally (or a previous push failed) — the next sync
	// should still push it even though there's nothing new to commit.
	skipIfNoGit(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644)
	if _, err := Commit(dataDir, "before remote existed"); err != nil {
		t.Fatalf("initial Commit: %v", err)
	}
	addOrigin(t, dataDir)

	result, err := Commit(dataDir, "")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if result.Committed {
		t.Fatalf("expected no new commit on a clean tree, got %+v", result)
	}
	if !result.Pushed || result.PushError != nil {
		t.Fatalf("expected the earlier commit to get pushed anyway, got %+v", result)
	}
}

func TestCommitReportsPushErrorWithoutFailing(t *testing.T) {
	skipIfNoGit(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "manifest.toml"), []byte("hello"), 0o644)
	if err := ensureRepo(dataDir); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "remote", "add", "origin", "/nonexistent/path/to/nowhere.git")
	cmd.Dir = dataDir
	cmd.Run()

	result, err := Commit(dataDir, "push will fail")
	if err != nil {
		t.Fatalf("expected Commit to succeed locally despite a broken remote, got error: %v", err)
	}
	if !result.Committed {
		t.Fatalf("expected the local commit to still succeed: %+v", result)
	}
	if result.Pushed || result.PushError == nil {
		t.Fatalf("expected a reported push error, got %+v", result)
	}
}
