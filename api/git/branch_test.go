package git

import (
	"os/exec"
	"testing"
)

func TestGitCreateNewBranchBasedOnTagAndSwitch(t *testing.T) {
	_, gitBranch, _ := setupTestGitRepoWithTag(t, "v1.2.3")

	newBranch := "feature-from-tag"
	gitBranch.GitCreateNewBranchBasedOnTagAndSwitch(newBranch, "v1.2.3")

	gitBranch.GetLatestBranchesInfo()
	current := gitBranch.CurrentCheckOut()
	if current.BranchName != newBranch {
		t.Errorf("expected current branch to be %s, got %s", newBranch, current.BranchName)
	}
}

func TestGitCreateNewBranchBasedOnTagAndSwitch_NonExistentTag(t *testing.T) {
	_, gitBranch, _ := setupTestGitRepoWithTag(t, "v1.2.3")

	gitBranch.GetLatestBranchesInfo()
	initialBranch := gitBranch.CurrentCheckOut().BranchName
	gitBranch.GitCreateNewBranchBasedOnTagAndSwitch("invalid-branch", "nonexistent-tag")

	gitBranch.GetLatestBranchesInfo()
	current := gitBranch.CurrentCheckOut()
	if current.BranchName != initialBranch {
		t.Errorf("expected current branch to remain %s, got %s", initialBranch, current.BranchName)
	}
}

func TestGitCreateNewBranchBasedOnTagAndSwitch_AmbiguousBranchName(t *testing.T) {
	_, gitBranch, tempDir := setupTestGitRepoWithTag(t, "v1.2.3")

	// Create an ambiguous branch named tags/v1.2.3
	cmdBranch := exec.Command("git", "branch", "tags/v1.2.3")
	cmdBranch.Dir = tempDir
	if err := cmdBranch.Run(); err != nil {
		t.Fatalf("git branch failed: %v", err)
	}

	newBranch := "branch-from-ambiguous-tag"
	gitBranch.GitCreateNewBranchBasedOnTagAndSwitch(newBranch, "v1.2.3")

	gitBranch.GetLatestBranchesInfo()
	current := gitBranch.CurrentCheckOut()
	if current.BranchName != newBranch {
		t.Errorf("expected current branch to be %s, got %s", newBranch, current.BranchName)
	}
}
