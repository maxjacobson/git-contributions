package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// headerRe matches the log format line for one commit:
// "2024-03;;;Alice;;;abc123 def456", where the parents are empty for root
// commits.
var headerRe = regexp.MustCompile(`^(\d{4}-\d{2});;;(.*);;;(.*)$`)

var (
	insertionsRe = regexp.MustCompile(`(\d+) insertion`)
	deletionsRe  = regexp.MustCompile(`(\d+) deletion`)
)

// ParseLog parses the output of
// git log --pretty=tformat:"%ad;;;%aN;;;%P" --date=format:"%Y-%m" --shortstat
// into commits. Commits with no stat line and fewer than two parents are
// skipped as empty, since a merge commit has no stat but is a real commit,
// while an empty commit is not a contribution.
func ParseLog(output string) ([]Commit, error) {
	var commits []Commit
	current := Commit{}
	started := false
	hasStat := false

	flush := func() {
		if started && (hasStat || parentCount(current) >= 2) {
			commits = append(commits, current)
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if matches := headerRe.FindStringSubmatch(line); matches != nil {
			flush()
			current = Commit{Month: matches[1], Author: matches[2], Parents: matches[3]}
			started = true
			hasStat = false
			continue
		}

		if !started {
			if strings.TrimSpace(line) == "" {
				continue
			}
			return nil, fmt.Errorf("unexpected log line before first commit: %q", line)
		}

		if strings.TrimSpace(line) == "" {
			continue
		}
		current.Insertions = countMatches(insertionsRe, line)
		current.Deletions = countMatches(deletionsRe, line)
		hasStat = true
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()
	return commits, nil
}

func parentCount(c Commit) int {
	return len(strings.Fields(c.Parents))
}

// countMatches returns the first captured number, or 0 if there is none,
// since a commit may touch only one side or nothing at all.
func countMatches(re *regexp.Regexp, line string) int {
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return 0
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return count
}
