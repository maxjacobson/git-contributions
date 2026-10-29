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
	// UseAll walks all refs, needed because jj's internal git repo has no
	// meaningful HEAD for a plain `git log`.
	UseAll bool
}

// DetectRepo checks whether we are inside a jj workspace and, if so, finds
// the underlying git directory with `jj git root`.
func DetectRepo() *Repo {
	if out, err := exec.Command("jj", "git", "root").Output(); err == nil {
		gitDir := strings.TrimSpace(string(out))
		if gitDir != "" {
			return &Repo{GitDir: gitDir, UseAll: true}
		}
	}
	return &Repo{}
}

// LogOutput runs git log and returns its raw output.
func (r *Repo) LogOutput() (string, error) {
	args := []string{"log", "--pretty=tformat:%ad;;;%aN;;;", "--date=format:%Y-%m", "--shortstat"}
	if r.UseAll {
		args = append(args, "--all")
	}
	if r.GitDir != "" {
		args = append([]string{"--git-dir", r.GitDir}, args...)
	}

	out, err := exec.Command("git", args...).Output()
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
