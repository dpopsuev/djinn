// commands_workspace.go — workspace management slash commands.
package repl

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/workspace"
)

// Workspace command names.
const (
	cmdWorkspace       = "/workspace"
	cmdWorkspaceSwitch = "/workspace-switch"
	cmdWorkspaceAdd    = "/workspace-add"
	cmdWorkspaceRepos  = "/workspace-repos"
	cmdWorkspaceSave   = "/workspace-save"
)

func executeWorkspaceSwitch(cmd Command, sess *session.Session) CommandResult {
	if len(cmd.Args) < 1 {
		return CommandResult{Output: "usage: /workspace-switch <name|file>"}
	}

	name := cmd.Args[0]
	newWS, err := workspace.Load(name)
	if err != nil {
		return CommandResult{Output: fmt.Sprintf("cannot load workspace %q: %v", name, err)}
	}

	oldName := sess.Workspace
	sess.Workspace = newWS.Name
	sess.WorkDirs = newWS.Paths()

	if globalWorkspaceBus != nil {
		var oldWS *workspace.Workspace
		if oldName != "" {
			oldWS, _ = workspace.Load(oldName)
		}
		globalWorkspaceBus.Emit(workspace.Event{
			Type: workspace.EventSwitch,
			Old:  oldWS,
			New:  newWS,
		})
	}

	return CommandResult{Output: fmt.Sprintf("switched to workspace %q (%d repos)", newWS.Name, len(newWS.Repos))}
}

func executeWorkspaceAdd(cmd Command, sess *session.Session) CommandResult {
	if len(cmd.Args) < 1 {
		return CommandResult{Output: "usage: /workspace-add <path>"}
	}
	dir := cmd.Args[0]
	sess.WorkDirs = append(sess.WorkDirs, dir)
	if globalWorkspaceBus != nil {
		globalWorkspaceBus.Emit(workspace.Event{
			Type: workspace.EventRepoAdd,
			New:  &workspace.Workspace{Repos: []workspace.Repo{{Path: dir}}},
			Repo: &workspace.Repo{Path: dir, Role: "dependency"},
		})
	}
	return CommandResult{Output: fmt.Sprintf("added repo: %s", dir)}
}

func executeWorkspaceRepos(sess *session.Session) CommandResult {
	if len(sess.WorkDirs) == 0 {
		return CommandResult{Output: "no repos in workspace"}
	}
	var sb strings.Builder
	for _, d := range sess.WorkDirs {
		fmt.Fprintf(&sb, "  %s\n", d)
	}
	return CommandResult{Output: strings.TrimRight(sb.String(), "\n")}
}

func executeWorkspaceSave(sess *session.Session) CommandResult {
	if sess.Workspace == "" {
		return CommandResult{Output: "name this workspace first: /workspace save <name>"}
	}
	return CommandResult{Output: fmt.Sprintf("workspace %q saved (use djinn workspace list to verify)", sess.Workspace)}
}

func executeWorkspace(cmd Command, sess *session.Session) CommandResult {
	if len(cmd.Args) == 0 {
		wsName := sess.Workspace
		if wsName == "" {
			wsName = "(ephemeral)"
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "Workspace: %s\n", wsName)
		if len(sess.WorkDirs) > 0 {
			fmt.Fprintf(&sb, "Repos:\n")
			for _, d := range sess.WorkDirs {
				fmt.Fprintf(&sb, "  %s\n", d)
			}
		}
		return CommandResult{Output: strings.TrimRight(sb.String(), "\n")}
	}

	switch cmd.Args[0] {
	case "repos":
		return executeWorkspaceRepos(sess)
	case "add":
		return executeWorkspaceAdd(Command{Args: cmd.Args[1:]}, sess)
	case "save":
		return executeWorkspaceSave(sess)
	default:
		return CommandResult{Output: "usage: /workspace [repos|add <path>|save]"}
	}
}
