package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
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
	// MailmapFile is the project's .mailmap, passed explicitly because git
	// does not discover it when --git-dir points inside .jj.
	MailmapFile string
	// StartCommit is the commit whose ancestry to walk. Empty means HEAD,
	// the plain git behavior.
	StartCommit string
	// IsJJ marks repos detected as jj workspaces.
	IsJJ bool
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
		repo := &Repo{GitDir: gitDir, IsJJ: true}
		if out, err := exec.Command("jj", "workspace", "root").Output(); err == nil {
			repo.Dir = strings.TrimSpace(string(out))
			if repo.Dir != "" {
				mailmap := repo.Dir + "/.mailmap"
				if _, err := os.Stat(mailmap); err == nil {
					repo.MailmapFile = mailmap
				}
			}
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
	args := r.gitArgs("log")
	if r.StartCommit != "" {
		args = append(args, r.StartCommit)
	}
	args = append(args, "--pretty=tformat:%ad;;;%aN;;;%H;;;%P;;;%s", "--date=format:%Y-%m-%d", "--shortstat")

	out, err := r.runGit(args...)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git log failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git log failed: %w", err)
	}
	return string(out), nil
}

// gitArgs builds a git command line with the repo's global options.
func (r *Repo) gitArgs(args ...string) []string {
	full := []string{}
	if r.MailmapFile != "" {
		full = append(full, "-c", "mailmap.file="+r.MailmapFile)
	}
	if r.GitDir != "" {
		full = append(full, "--git-dir", r.GitDir)
	}
	return append(full, args...)
}

func (r *Repo) runGit(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	return cmd.Output()
}

// GitHubSlug returns "owner/repo" if the origin remote points at
// github.com, or an empty string otherwise.
func (r *Repo) GitHubSlug() string {
	out, err := r.runGit(r.gitArgs("remote", "get-url", "origin")...)
	if err != nil {
		return ""
	}
	return ParseGitHubSlug(strings.TrimSpace(string(out)))
}

// ParseGitHubSlug extracts "owner/repo" from a github.com remote URL in
// SSH or HTTPS form.
func ParseGitHubSlug(url string) string {
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(\.git)?$`),
		regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(\.git)?$`),
		regexp.MustCompile(`^ssh://git@github\.com/([^/]+)/([^/]+?)(\.git)?$`),
	} {
		if matches := re.FindStringSubmatch(url); matches != nil {
			return matches[1] + "/" + matches[2]
		}
	}
	return ""
}

// ShowCommand returns the command that shows one commit's diff in our own
// always-paging less, since the user's pager may be "cat" (jj) or exit on
// short diffs (less -F). Delta, when installed, provides the syntax
// highlighting: it consumes the raw diff from either tool.

// shellQuote single-quotes a string for safe use in a shell command.

// LoadCommits fetches and parses the repo's commit history.
func (r *Repo) LoadCommits() ([]Commit, error) {
	output, err := r.LogOutput()
	if err != nil {
		return nil, err
	}
	return ParseLog(output)
}

// DiffOutput returns the colorized output of showing one commit. Delta, when
// configured, colorizes the diff with syntax highlighting (with no pager, so
// the output lands in our own viewport), using the invocation from the
// user's git pager config so themes and options carry over, and padded to
// the given width so line backgrounds run edge to edge.
func (r *Repo) DiffOutput(hash string, width int) (string, error) {
	var show string
	if r.IsJJ {
		show = fmt.Sprintf("jj show -r %s --no-pager --color never", hash)
	} else {
		show = "git --no-pager show " + hash
	}

	var script string
	if delta := r.deltaCommand(); delta != "" {
		if width > 0 {
			delta += fmt.Sprintf(" --width=%d", width)
		}
		script = show + " | " + delta
	} else if r.IsJJ {
		script = show + " --color always"
	} else {
		script = "git --no-pager -c color.ui=always show " + hash
	}

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = r.Dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("diff failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("diff failed: %w", err)
	}
	return string(out), nil
}

// deltaCommand returns the user's configured delta invocation from their git
// pager config, so their theme and options apply, or a bare delta if it is
// installed, or empty when delta is not in play.
func (r *Repo) deltaCommand() string {
	out, err := r.runGit(r.gitArgs("config", "--get", "core.pager")...)
	if err == nil {
		if pager := strings.TrimSpace(string(out)); strings.HasPrefix(pager, "delta") {
			return pager
		}
	}
	if _, err := exec.LookPath("delta"); err == nil {
		return "delta"
	}
	return ""
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
