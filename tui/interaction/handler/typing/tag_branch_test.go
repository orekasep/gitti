package typing

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/gohyuhan/gitti/api"
	"github.com/gohyuhan/gitti/api/git"
	"github.com/gohyuhan/gitti/executor"
	"github.com/gohyuhan/gitti/i18n"
	"github.com/gohyuhan/gitti/logging"
	"github.com/gohyuhan/gitti/tui/constant"
	branchPopUp "github.com/gohyuhan/gitti/tui/popup/branch"
	"github.com/gohyuhan/gitti/tui/types"
)

func init() {
	i18n.InitGittiLanguageMapping("EN")
}

func TestTypingEnter_CreateNewBranchBasedOnTag(t *testing.T) {
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
	run("tag", "v1.0.0")

	executor.InitCmdExecutor(tempDir)
	logChan := make(chan string, 100)
	logger := logging.InitGittiLogging(100, logChan, 10)
	lock := git.InitGitProcessLock(logger)
	gitBranch := git.InitGitBranch(lock, true, logger)
	gitOps := &api.GitOperations{
		GitBranch: gitBranch,
	}

	input := textinput.New()
	input.SetValue("new-tag-branch")

	popUpModel := &branchPopUp.CreateNewBranchPopUpModel{
		NewBranchNameInput: input,
		CreateType:         git.NEWBRANCHBASEDONTAG,
		BasedOnTagName:     "v1.0.0",
	}

	m := &types.GittiModel{
		PopUpType:     constant.CreateNewBranchPopUp,
		PopUpModel:    popUpModel,
		GitOperations: gitOps,
	}
	m.ShowPopUp.Store(true)
	m.IsTyping.Store(true)

	resModel, _ := handleTypingEnterKeyBindingInteraction(m, tea.KeyPressMsg{})

	if resModel.ShowPopUp.Load() {
		t.Errorf("expected ShowPopUp to be false after submission")
	}
	if resModel.IsTyping.Load() {
		t.Errorf("expected IsTyping to be false after submission")
	}
	if resModel.PopUpType != constant.NoPopUp {
		t.Errorf("expected PopUpType to be NoPopUp, got %s", resModel.PopUpType)
	}
	if resModel.PopUpModel != nil {
		t.Errorf("expected PopUpModel to be nil, got %v", resModel.PopUpModel)
	}

	// Assert side effect: new branch is created based on tag and checked out
	for i := 0; i < 50; i++ {
		gitBranch.GetLatestBranchesInfo()
		if gitBranch.CurrentCheckOut().BranchName == "new-tag-branch" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	current := gitBranch.CurrentCheckOut()
	if current.BranchName != "new-tag-branch" {
		t.Errorf("expected current branch to be 'new-tag-branch', got '%s'", current.BranchName)
	}
}
