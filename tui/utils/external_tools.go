package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ------------------------------------
//
//	EditorPreset holds metadata for standard editor presets.
//
// ------------------------------------
type EditorPreset struct {
	Command      string
	IsGUI        bool
	LineArgStyle string // "+line", "-g", "file:line"
}

// EditorPresets provides presets for popular terminal and GUI editors.
var EditorPresets = map[string]EditorPreset{
	"vim":         {Command: "vim", IsGUI: false, LineArgStyle: "+line"},
	"vi":          {Command: "vi", IsGUI: false, LineArgStyle: "+line"},
	"nvim":        {Command: "nvim", IsGUI: false, LineArgStyle: "+line"},
	"neovim":      {Command: "nvim", IsGUI: false, LineArgStyle: "+line"},
	"nano":        {Command: "nano", IsGUI: false, LineArgStyle: "+line"},
	"emacs":       {Command: "emacs", IsGUI: false, LineArgStyle: "+line"},
	"micro":       {Command: "micro", IsGUI: false, LineArgStyle: "+line"},
	"helix":       {Command: "hx", IsGUI: false, LineArgStyle: "file:line"},
	"hx":          {Command: "hx", IsGUI: false, LineArgStyle: "file:line"},
	"vscode":      {Command: "code", IsGUI: true, LineArgStyle: "-g"},
	"code":        {Command: "code", IsGUI: true, LineArgStyle: "-g"},
	"cursor":      {Command: "cursor", IsGUI: true, LineArgStyle: "-g"},
	"zed":         {Command: "zed", IsGUI: true, LineArgStyle: "file:line"},
	"windsurf":    {Command: "windsurf", IsGUI: true, LineArgStyle: "-g"},
	"antigravity": {Command: "antigravity", IsGUI: true, LineArgStyle: "-g"},
	"sublime":     {Command: "subl", IsGUI: true, LineArgStyle: "file:line"},
	"subl":        {Command: "subl", IsGUI: true, LineArgStyle: "file:line"},
}

func isKnownGUICommand(cmdName string) bool {
	switch strings.ToLower(cmdName) {
	case "code", "cursor", "zed", "windsurf", "antigravity", "subl", "sublime_text":
		return true
	default:
		return false
	}
}

// ------------------------------------
//
//	ResolveEditorCommand returns the command string and whether it's a GUI editor.
//	Fallback order: userSetEditor (if not auto/empty) -> $GIT_EDITOR -> $VISUAL -> $EDITOR -> "vim"
//
// ------------------------------------
func ResolveEditorCommand(userSetEditor string) (string, bool) {
	raw := strings.TrimSpace(userSetEditor)
	if raw == "" || strings.EqualFold(raw, "auto") {
		if gitEditor := os.Getenv("GIT_EDITOR"); gitEditor != "" {
			raw = strings.TrimSpace(gitEditor)
		} else if visual := os.Getenv("VISUAL"); visual != "" {
			raw = strings.TrimSpace(visual)
		} else if editor := os.Getenv("EDITOR"); editor != "" {
			raw = strings.TrimSpace(editor)
		} else {
			raw = "vim"
		}
	}

	if preset, ok := EditorPresets[strings.ToLower(raw)]; ok {
		return preset.Command, preset.IsGUI
	}

	parts := strings.Fields(raw)
	if len(parts) > 0 {
		return raw, isKnownGUICommand(parts[0])
	}
	return "vim", false
}

// ------------------------------------
//
//	ReturnEditorLaunchCommand creates a command to launch the specified editor with the given file.
//
// ------------------------------------
func ReturnEditorLaunchCommand(fileName string, userSetEditor string) (*exec.Cmd, bool) {
	return ReturnEditorLaunchCommandWithLine(fileName, 0, userSetEditor)
}

// ------------------------------------
//
//	ReturnEditorLaunchCommandWithLine creates an exec.Cmd to launch the editor at an optional line number.
//
// ------------------------------------
func ReturnEditorLaunchCommandWithLine(fileName string, lineNum int, userSetEditor string) (*exec.Cmd, bool) {
	filepath := "."
	if fileName != "" {
		filepath = fileName
	}

	rawCmd, isGUI := ResolveEditorCommand(userSetEditor)
	parts := strings.Fields(rawCmd)
	if len(parts) == 0 {
		parts = []string{"vim"}
	}

	bin := parts[0]
	args := parts[1:]

	hasFileVar := false
	for i, arg := range args {
		if strings.Contains(arg, "{{file}}") || strings.Contains(arg, "{{filename}}") {
			hasFileVar = true
			arg = strings.ReplaceAll(arg, "{{file}}", filepath)
			arg = strings.ReplaceAll(arg, "{{filename}}", filepath)
			if lineNum > 0 {
				arg = strings.ReplaceAll(arg, "{{line}}", strconv.Itoa(lineNum))
			} else {
				arg = strings.ReplaceAll(arg, "{{line}}", "1")
			}
			args[i] = arg
		}
	}

	if !hasFileVar {
		preset, isPreset := EditorPresets[strings.ToLower(bin)]
		if lineNum > 0 {
			if isPreset {
				switch preset.LineArgStyle {
				case "+line":
					args = append(args, fmt.Sprintf("+%d", lineNum), filepath)
				case "-g":
					args = append(args, "-g", fmt.Sprintf("%s:%d", filepath, lineNum))
				case "file:line":
					args = append(args, fmt.Sprintf("%s:%d", filepath, lineNum))
				default:
					args = append(args, filepath)
				}
			} else {
				args = append(args, fmt.Sprintf("+%d", lineNum), filepath)
			}
		} else {
			args = append(args, filepath)
		}
	}

	cmd := exec.Command(bin, args...)
	return cmd, isGUI
}

// ------------------------------------
//
//	BuildDiffViewerCommand builds the exec.Cmd for the given viewer, target file,
//	staged state, commit hash, or stash ref.
//
// ------------------------------------
func BuildDiffViewerCommand(viewer string, filePath string, isStaged bool, commitHash string, stashRef string) (*exec.Cmd, bool) {
	trimmedViewer := strings.TrimSpace(viewer)
	if trimmedViewer == "" || strings.EqualFold(trimmedViewer, "auto") {
		if _, err := exec.LookPath("hunk"); err == nil {
			trimmedViewer = "hunk"
		} else if _, err := exec.LookPath("delta"); err == nil {
			trimmedViewer = "delta"
		} else {
			trimmedViewer = "git"
		}
	}

	if strings.Contains(trimmedViewer, "{{file}}") || strings.Contains(trimmedViewer, "{{hash}}") || strings.Contains(trimmedViewer, "{{stash}}") {
		cmdStr := strings.ReplaceAll(trimmedViewer, "{{file}}", filePath)
		cmdStr = strings.ReplaceAll(cmdStr, "{{hash}}", commitHash)
		cmdStr = strings.ReplaceAll(cmdStr, "{{stash}}", stashRef)
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			return nil, false
		}
		return exec.Command(parts[0], parts[1:]...), true
	}

	switch strings.ToLower(trimmedViewer) {
	case "hunk":
		if commitHash != "" {
			return exec.Command("hunk", "show", commitHash), true
		}
		if stashRef != "" {
			return exec.Command("hunk", "stash", "show", stashRef), true
		}
		if filePath != "" {
			if isStaged {
				return exec.Command("hunk", "diff", "--staged", "--", filePath), true
			}
			return exec.Command("hunk", "diff", "--", filePath), true
		}
		return exec.Command("hunk", "diff"), true

	case "delta":
		if commitHash != "" {
			return exec.Command("git", "-c", "core.pager=delta", "show", commitHash), true
		}
		if stashRef != "" {
			return exec.Command("git", "-c", "core.pager=delta", "stash", "show", "-p", stashRef), true
		}
		if filePath != "" {
			if isStaged {
				return exec.Command("git", "-c", "core.pager=delta", "diff", "--staged", "--", filePath), true
			}
			return exec.Command("git", "-c", "core.pager=delta", "diff", "--", filePath), true
		}
		return exec.Command("git", "-c", "core.pager=delta", "diff"), true

	case "git":
		if commitHash != "" {
			return exec.Command("git", "show", commitHash), true
		}
		if stashRef != "" {
			return exec.Command("git", "stash", "show", "-p", stashRef), true
		}
		if filePath != "" {
			if isStaged {
				return exec.Command("git", "diff", "--staged", "--", filePath), true
			}
			return exec.Command("git", "diff", "--", filePath), true
		}
		return exec.Command("git", "diff"), true

	default:
		parts := strings.Fields(trimmedViewer)
		if len(parts) == 0 {
			return nil, false
		}
		bin := parts[0]
		args := parts[1:]
		if commitHash != "" {
			args = append(args, commitHash)
		} else if stashRef != "" {
			args = append(args, stashRef)
		} else if filePath != "" {
			args = append(args, filePath)
		}
		return exec.Command(bin, args...), true
	}
}
