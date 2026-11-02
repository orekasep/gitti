package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gohyuhan/gitti/executor"
	"github.com/gohyuhan/gitti/i18n"
	"github.com/gohyuhan/gitti/logging"
)

func setupTestGitRepoWithTag(t *testing.T, tagName string) (*GitTag, *GitBranch, string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	tempDir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s", args, string(out))
		}
	}

	run("init")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.com")

	dummyFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(dummyFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	run("add", "test.txt")
	run("commit", "-m", "initial commit")
	run("tag", tagName)

	i18n.InitGittiLanguageMapping("EN")
	executor.InitCmdExecutor(tempDir)
	logChan := make(chan string, 100)
	logger := logging.InitGittiLogging(100, logChan, 10)
	lock := InitGitProcessLock(logger)
	gitTag := InitGitTag(logChan, lock, logger)
	gitBranch := InitGitBranch(lock, true, logger)

	return gitTag, gitBranch, tempDir
}

func TestGitCheckoutTag(t *testing.T) {
	gitTag, _, tempDir := setupTestGitRepoWithTag(t, "v1.0.0")

	output, success := gitTag.GitCheckoutTag("v1.0.0")
	if !success {
		t.Fatalf("GitCheckoutTag failed, output: %v", output)
	}

	// Verify HEAD is detached: git symbolic-ref -q HEAD fails
	symCmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	symCmd.Dir = tempDir
	if err := symCmd.Run(); err == nil {
		t.Errorf("expected symbolic-ref to fail for detached HEAD, but succeeded")
	}

	// Verify HEAD points to the tag commit
	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headCmd.Dir = tempDir
	headOut, err := headCmd.Output()
	if err != nil {
		t.Fatalf("failed to rev-parse HEAD: %v", err)
	}

	tagCmd := exec.Command("git", "rev-parse", "v1.0.0^{commit}")
	tagCmd.Dir = tempDir
	tagOut, err := tagCmd.Output()
	if err != nil {
		t.Fatalf("failed to rev-parse tag commit: %v", err)
	}

	if string(headOut) != string(tagOut) {
		t.Errorf("expected HEAD (%s) to match tag commit (%s)", string(headOut), string(tagOut))
	}
}

func TestGitCheckoutTag_AmbiguousBranchName(t *testing.T) {
	gitTag, _, tempDir := setupTestGitRepoWithTag(t, "v1.0.0")

	// Create an ambiguous branch named tags/v1.0.0 on a second commit
	dummyFile := filepath.Join(tempDir, "dummy.txt")
	if err := os.WriteFile(dummyFile, []byte("second commit\n"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
	cmdAdd := exec.Command("git", "add", "dummy.txt")
	cmdAdd.Dir = tempDir
	if err := cmdAdd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	cmdCommit := exec.Command("git", "commit", "-m", "second commit")
	cmdCommit.Dir = tempDir
	if err := cmdCommit.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	cmdBranch := exec.Command("git", "branch", "tags/v1.0.0")
	cmdBranch.Dir = tempDir
	if err := cmdBranch.Run(); err != nil {
		t.Fatalf("git branch failed: %v", err)
	}

	output, success := gitTag.GitCheckoutTag("v1.0.0")
	if !success {
		t.Fatalf("GitCheckoutTag failed, output: %v", output)
	}

	// Must be on tag commit (detached HEAD), not on branch tags/v1.0.0
	tagCmd := exec.Command("git", "rev-parse", "v1.0.0^{commit}")
	tagCmd.Dir = tempDir
	tagOut, err := tagCmd.Output()
	if err != nil {
		t.Fatalf("failed to rev-parse tag commit: %v", err)
	}

	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headCmd.Dir = tempDir
	headOut, err := headCmd.Output()
	if err != nil {
		t.Fatalf("failed to rev-parse HEAD: %v", err)
	}

	if string(headOut) != string(tagOut) {
		t.Errorf("expected HEAD to match tag commit, got %s vs %s", string(headOut), string(tagOut))
	}
}

func TestGitCheckoutTag_NonExistent(t *testing.T) {
	gitTag, _, _ := setupTestGitRepoWithTag(t, "v1.0.0")

	output, success := gitTag.GitCheckoutTag("v9.9.9")
	if success {
		t.Fatalf("expected GitCheckoutTag to fail for nonexistent tag, got success with output: %v", output)
	}
}
