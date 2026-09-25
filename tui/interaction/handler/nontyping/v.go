package nontyping

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/gohyuhan/gitti/tui/component/commitlog"
	"github.com/gohyuhan/gitti/tui/component/files"
	"github.com/gohyuhan/gitti/tui/component/reflog"
	"github.com/gohyuhan/gitti/tui/component/stash"
	"github.com/gohyuhan/gitti/tui/constant"
	"github.com/gohyuhan/gitti/tui/types"
	"github.com/gohyuhan/gitti/tui/utils"
)

// ------------------------------------
//
//	Handle 'v' key interaction.
//	Responsibility: Contextual "diff view" action.
//	Launches the configured external diff viewer (such as hunk) for the currently
//	selected file (staged/unstaged), commit log, or stash entry.
//
// ------------------------------------
func handleNonTypingvKeyBindingInteraction(m *types.GittiModel) (*types.GittiModel, tea.Cmd) {
	if m.ShowPopUp.Load() || m.IsLineEditingState.Load() || m.IsPanelFiltering.Load() {
		return m, nil
	}

	cmd, ok := returnDiffViewerLaunchCommand(m)
	if !ok || cmd == nil {
		return m, nil
	}

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return types.DiffViewerFinishedMsg{
			Err: err,
		}
	})
}

func returnDiffViewerLaunchCommand(m *types.GittiModel) (*exec.Cmd, bool) {
	viewer := m.UserSetDiffViewer

	switch m.CurrentSelectedComponent {
	case constant.ModifiedFilesComponentPanel:
		selectedItem := m.CurrentRepoModifiedFilesInfoList.SelectedItem()
		if selectedItem == nil {
			return nil, false
		}
		file := selectedItem.(files.GitModifiedFilesItem)
		isStaged := file.IndexState != " " && file.WorkTree == " "
		return utils.BuildDiffViewerCommand(viewer, file.FilePathname, isStaged, "", "")

	case constant.DetailComponentPanel, constant.DetailComponentPanelTwo:
		if m.DetailPanelParentComponent == constant.ModifiedFilesComponentPanel {
			selectedItem := m.CurrentRepoModifiedFilesInfoList.SelectedItem()
			if selectedItem == nil {
				return nil, false
			}
			file := selectedItem.(files.GitModifiedFilesItem)
			isStaged := false
			if m.CurrentSelectedComponent == constant.DetailComponentPanelTwo {
				isStaged = false
			} else if file.IndexState != " " && file.WorkTree != " " {
				isStaged = true
			} else if file.IndexState != " " {
				isStaged = true
			}
			return utils.BuildDiffViewerCommand(viewer, file.FilePathname, isStaged, "", "")
		} else if m.DetailPanelParentComponent == constant.CommitLogOrRefLogComponentPanel {
			if m.CurrentCommitLogOrRefLogComponentShowing == constant.SHOW_COMMITLOG {
				if item := m.CurrentRepoCommitLogInfoList.SelectedItem(); item != nil {
					commitItem := item.(commitlog.GitCommitLogItem)
					return utils.BuildDiffViewerCommand(viewer, "", false, commitItem.Hash, "")
				}
			} else if m.CurrentCommitLogOrRefLogComponentShowing == constant.SHOW_REFLOG {
				if item := m.CurrentRepoRefLogInfoList.SelectedItem(); item != nil {
					reflogItem := item.(reflog.GitRefLogItem)
					return utils.BuildDiffViewerCommand(viewer, "", false, reflogItem.Hash, "")
				}
			}
		} else if m.DetailPanelParentComponent == constant.StashComponentPanel {
			if item := m.CurrentRepoStashInfoList.SelectedItem(); item != nil {
				stashItem := item.(stash.GitStashItem)
				return utils.BuildDiffViewerCommand(viewer, "", false, "", stashItem.Id)
			}
		}

	case constant.CommitLogOrRefLogComponentPanel:
		if m.CurrentCommitLogOrRefLogComponentShowing == constant.SHOW_COMMITLOG {
			if item := m.CurrentRepoCommitLogInfoList.SelectedItem(); item != nil {
				commitItem := item.(commitlog.GitCommitLogItem)
				return utils.BuildDiffViewerCommand(viewer, "", false, commitItem.Hash, "")
			}
		} else if m.CurrentCommitLogOrRefLogComponentShowing == constant.SHOW_REFLOG {
			if item := m.CurrentRepoRefLogInfoList.SelectedItem(); item != nil {
				reflogItem := item.(reflog.GitRefLogItem)
				return utils.BuildDiffViewerCommand(viewer, "", false, reflogItem.Hash, "")
			}
		}

	case constant.StashComponentPanel:
		if item := m.CurrentRepoStashInfoList.SelectedItem(); item != nil {
			stashItem := item.(stash.GitStashItem)
			return utils.BuildDiffViewerCommand(viewer, "", false, "", stashItem.Id)
		}
	}

	return nil, false
}
