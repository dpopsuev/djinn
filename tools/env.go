package tools

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	misbahDefaultSocket = "/run/misbah/permission.sock"
	gitBranch           = "git"
	gitBranchArgs       = "rev-parse --abbrev-ref HEAD"
	gitStatusArgs       = "status --porcelain"
)

// EnvSnapshot captures the current environment state in one call.
type EnvSnapshot struct {
	GitBranch      string `json:"git_branch"`
	UncommittedFiles int  `json:"uncommitted_files"`
	GoVersion      string `json:"go_version"`
	Hostname       string `json:"hostname"`
	WorkingDir     string `json:"working_dir"`
	MisbahDaemon   bool   `json:"misbah_daemon"`
}

// Snapshot captures the current environment state.
func Snapshot() EnvSnapshot {
	snap := EnvSnapshot{
		GitBranch:      gitCurrentBranch(),
		UncommittedFiles: gitUncommittedCount(),
		GoVersion:      goVersion(),
		WorkingDir:     workingDir(),
		Hostname:       hostname(),
		MisbahDaemon:   misbahReachable(),
	}
	return snap
}

func gitCurrentBranch() string {
	out, err := exec.Command(gitBranch, strings.Fields(gitBranchArgs)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitUncommittedCount() int {
	out, err := exec.Command(gitBranch, strings.Fields(gitStatusArgs)...).Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

func goVersion() string {
	return runtime.Version()
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func workingDir() string {
	d, _ := os.Getwd()
	return d
}

func misbahReachable() bool {
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = misbahDefaultSocket
	}
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
