package nontyping

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	"github.com/gohyuhan/gitti/api"
	"github.com/gohyuhan/gitti/api/git"
	"github.com/gohyuhan/gitti/executor"
	"github.com/gohyuhan/gitti/i18n"
	"github.com/gohyuhan/gitti/logging"
	tagComponent "github.com/gohyuhan/gitti/tui/component/tag"
	"github.com/gohyuhan/gitti/tui/constant"
	branchPopUp "github.com/gohyuhan/gitti/tui/popup/branch"
	tagPopUp "github.com/gohyuhan/gitti/tui/popup/tag"
	"github.com/gohyuhan/gitti/tui/types"
)

func init() {
	i18n.InitGittiLanguageMapping("EN")
}

func setupTagModel(t *testing.T, tagName string) *types.GittiModel {
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
	if err := os.WriteFile(dummyFile, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	run("add", "test.txt")
	run("commit", "-m", "init")
	run("tag", tagName)

	executor.InitCmdExecutor(tempDir)
	logChan := make(chan string, 100)
	logger := logging.InitGittiLogging(100, logChan, 10)
	lock := git.InitGitProcessLock(logger)
	gitBranch := git.InitGitBranch(lock, true, logger)
	gitOps := &api.GitOperations{
		GitBranch: gitBranch,
	}

	tagItems := []list.Item{
		tagComponent.GitTagItem{
			TagName: tagName,
		},
	}
	tagList := list.New(tagItems, tagComponent.GitTagItemDelegate{}, 80, 20)

	m := &types.GittiModel{
		Width:                    100,
		Height:                   40,
		CurrentSelectedComponent: constant.LocalBranchOrTagOrRemoteOrWorktreeComponentPanel,
		CurrentLocalBranchOrTagOrRemoteOrWorktreeComponentShowing: constant.SHOW_TAG,
		CurrentRepoTagInfoList: tagList,
		TuiUpdateChannel:       make(chan interface{}, 10),
		PopUpType:              constant.NoPopUp,
		GitOperations:          gitOps,
	}
	m.ShowPopUp.Store(false)
	m.IsTyping.Store(false)
	return m
}

func TestTagEnter_OpensChooseTagActionPopUp(t *testing.T) {
	m := setupTagModel(t, "v1.0.0")

	resModel, _ := handleNonTypingEnterKeyBindingInteraction(m)

	if !resModel.ShowPopUp.Load() {
		t.Fatalf("expected ShowPopUp to be true")
	}
	if resModel.PopUpType != constant.ChooseTagActionPopUp {
		t.Fatalf("expected PopUpType to be %s, got %s", constant.ChooseTagActionPopUp, resModel.PopUpType)
	}
	popUp, ok := resModel.PopUpModel.(*tagPopUp.ChooseTagActionPopUpModel)
	if !ok {
		t.Fatalf("expected PopUpModel to be *tagPopUp.ChooseTagActionPopUpModel")
	}
	if popUp.TagName != "v1.0.0" {
		t.Errorf("expected TagName 'v1.0.0', got '%s'", popUp.TagName)
	}
	if len(popUp.TagActionList.Items()) != 2 {
		t.Fatalf("expected 2 options, got %d", len(popUp.TagActionList.Items()))
	}
}

func TestTagEnter_SelectCheckoutTagOption(t *testing.T) {
	m := setupTagModel(t, "v2.0.0")
	// Open choose tag action popup
	m, _ = handleNonTypingEnterKeyBindingInteraction(m)

	popUp := m.PopUpModel.(*tagPopUp.ChooseTagActionPopUpModel)
	popUp.TagActionList.Select(0) // Checkout option

	// Press enter on the option -> transitions to choose switch type popup
	resModel, _ := handleNonTypingEnterKeyBindingInteraction(m)

	if resModel.PopUpType != constant.ChooseSwitchBranchTypePopUp {
		t.Fatalf("expected PopUpType to be %s, got %s", constant.ChooseSwitchBranchTypePopUp, resModel.PopUpType)
	}
	switchPopUp, ok := resModel.PopUpModel.(*branchPopUp.ChooseSwitchBranchTypePopUpModel)
	if !ok {
		t.Fatalf("expected PopUpModel to be *branchPopUp.ChooseSwitchBranchTypePopUpModel")
	}
	if switchPopUp.BranchName != "refs/tags/v2.0.0" {
		t.Errorf("expected BranchName 'refs/tags/v2.0.0', got '%s'", switchPopUp.BranchName)
	}

	// Press enter on switch type option -> transitions to SwitchBranchOutputPopUp
	switchPopUp.SwitchTypeOptionList.Select(0)
	resModel, cmd := handleNonTypingEnterKeyBindingInteraction(resModel)
	if resModel.PopUpType != constant.SwitchBranchOutputPopUp {
		t.Fatalf("expected PopUpType to be %s, got %s", constant.SwitchBranchOutputPopUp, resModel.PopUpType)
	}
	outputPopUp, ok := resModel.PopUpModel.(*branchPopUp.SwitchBranchOutputPopUpModel)
	if !ok {
		t.Fatalf("expected PopUpModel to be *branchPopUp.SwitchBranchOutputPopUpModel")
	}
	if outputPopUp.BranchName != "refs/tags/v2.0.0" {
		t.Errorf("expected BranchName 'refs/tags/v2.0.0', got '%s'", outputPopUp.BranchName)
	}
	if !outputPopUp.IsProcessing.Load() {
		t.Errorf("expected IsProcessing to be true")
	}
	if cmd == nil {
		t.Errorf("expected spinner tick cmd to be returned")
	}

	// Wait for switch branch result event from channel
	select {
	case msg := <-m.TuiUpdateChannel:
		updateMsg, ok := msg.(types.GittiTuiUpdateMsg)
		if !ok {
			t.Fatalf("expected GittiTuiUpdateMsg, got %T", msg)
		}
		if updateMsg.Event != constant.GIT_SWITCH_BRANCH_RESULT_EVENT {
			t.Errorf("expected event %s, got %s", constant.GIT_SWITCH_BRANCH_RESULT_EVENT, updateMsg.Event)
		}
		data, ok := updateMsg.Data.(types.GitSwitchBranchResultEventDataStructure)
		if !ok || !data.Success {
			t.Errorf("expected switch branch to succeed, got %v", updateMsg.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for switch branch result event")
	}
}

func TestTagEnter_SelectCreateBranchOption(t *testing.T) {
	m := setupTagModel(t, "v3.0.0")
	// Open choose tag action popup
	m, _ = handleNonTypingEnterKeyBindingInteraction(m)

	popUp := m.PopUpModel.(*tagPopUp.ChooseTagActionPopUpModel)
	popUp.TagActionList.Select(1) // Create branch option

	// Press enter on the option
	resModel, _ := handleNonTypingEnterKeyBindingInteraction(m)

	if resModel.PopUpType != constant.CreateNewBranchPopUp {
		t.Fatalf("expected PopUpType to be %s, got %s", constant.CreateNewBranchPopUp, resModel.PopUpType)
	}
	if !resModel.IsTyping.Load() {
		t.Errorf("expected IsTyping to be true")
	}
	branchPopUpModel, ok := resModel.PopUpModel.(*branchPopUp.CreateNewBranchPopUpModel)
	if !ok {
		t.Fatalf("expected PopUpModel to be *branchPopUp.CreateNewBranchPopUpModel")
	}
	if branchPopUpModel.CreateType != git.NEWBRANCHBASEDONTAG {
		t.Errorf("expected CreateType %s, got %s", git.NEWBRANCHBASEDONTAG, branchPopUpModel.CreateType)
	}
	if branchPopUpModel.BasedOnTagName != "v3.0.0" {
		t.Errorf("expected BasedOnTagName 'v3.0.0', got '%s'", branchPopUpModel.BasedOnTagName)
	}
}

func TestTagNKey_DirectlyOpensCreateBranchPopUp(t *testing.T) {
	m := setupTagModel(t, "v4.0.0")

	resModel, _ := handleNonTypingnKeyBindingInteraction(m)

	if !resModel.ShowPopUp.Load() {
		t.Fatalf("expected ShowPopUp to be true")
	}
	if resModel.PopUpType != constant.CreateNewBranchPopUp {
		t.Fatalf("expected PopUpType to be %s, got %s", constant.CreateNewBranchPopUp, resModel.PopUpType)
	}
	if !resModel.IsTyping.Load() {
		t.Errorf("expected IsTyping to be true")
	}
	branchPopUpModel, ok := resModel.PopUpModel.(*branchPopUp.CreateNewBranchPopUpModel)
	if !ok {
		t.Fatalf("expected PopUpModel to be *branchPopUp.CreateNewBranchPopUpModel")
	}
	if branchPopUpModel.CreateType != git.NEWBRANCHBASEDONTAG {
		t.Errorf("expected CreateType %s, got %s", git.NEWBRANCHBASEDONTAG, branchPopUpModel.CreateType)
	}
	if branchPopUpModel.BasedOnTagName != "v4.0.0" {
		t.Errorf("expected BasedOnTagName 'v4.0.0', got '%s'", branchPopUpModel.BasedOnTagName)
	}
}
