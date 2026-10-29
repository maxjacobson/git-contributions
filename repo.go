package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// Repo knows how to fetch this project's commit history, whether the working
// directory is a plain git repo or a jj workspace.
type Repo struct {
	// GitDir is jj's underlying git directory, empty for plain git repos.
	GitDir string
	// Dir is the directory to run git from. For jj repos this is the
	// workspace root.
	Dir string
	// StartCommit is the commit whose ancestry to walk. Empty means HEAD,
	// the plain git behavior.
	StartCommit string
}

// DetectRepo checks whether we are inside a jj workspace and, if so, finds
// the underlying git directory and the working copy commit whose ancestry to
// walk.
func DetectRepo() *Repo {
	if out, err := exec.Command("jj", "git", "root").Output(); err == nil {
		gitDir := strings.TrimSpace(string(out))
		if gitDir == "" {
			return &Repo{}
		}
		repo := &Repo{GitDir: gitDir}
		if out, err := exec.Command("jj", "workspace", "root").Output(); err == nil {
			repo.Dir = strings.TrimSpace(string(out))
		}
		if out, err := exec.Command("jj", "log", "-r", "@", "--no-graph", "-T", `commit_id ++ "\n"`).Output(); err == nil {
			repo.StartCommit = strings.TrimSpace(string(out))
		}
		return repo
	}
	return &Repo{}
}

// LogOutput runs git log and returns its raw output.
func (r *Repo) LogOutput() (string, error) {
	args := []string{}
	if r.GitDir != "" {
		args = append(args, "--git-dir", r.GitDir)
	}
	args = append(args, "log")
	if r.StartCommit != "" {
		args = append(args, r.StartCommit)
	}
	args = append(args, "--pretty=tformat:%ad;;;%aN;;;%P", "--date=format:%Y-%m", "--shortstat")

	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git log failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git log failed: %w", err)
	}
	return string(out), nil
}

// LoadCommits fetches and parses the repo's commit history.
func (r *Repo) LoadCommits() ([]Commit, error) {
	output, err := r.LogOutput()
	if err != nil {
		return nil, err
	}
	return ParseLog(output)
}

// FilterDependabot drops commits by dependabot, matching the original
// script's behavior.
func FilterDependabot(commits []Commit) []Commit {
	kept := commits[:0]
	for _, c := range commits {
		if !strings.Contains(c.Author, "dependabot") {
			kept = append(kept, c)
		}
	}
	return kept
}
