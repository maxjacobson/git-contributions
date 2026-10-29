package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// dateRe matches the log format date, "2024-03-15".
var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var (
	insertionsRe = regexp.MustCompile(`(\d+) insertion`)
	deletionsRe  = regexp.MustCompile(`(\d+) deletion`)
)

// ParseLog parses the output of
// git log --pretty=tformat:"%ad;;;%aN;;;%H;;;%P;;;%s" --date=format:"%Y-%m-%d" --shortstat
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

		if parts, ok := parseHeader(line); ok {
			flush()
			current = parts
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

// parseHeader splits a header line into a Commit. The subject is everything
// after the fourth separator, so subjects containing ";;;" survive intact.
func parseHeader(line string) (Commit, bool) {
	parts := strings.SplitN(line, ";;;", 5)
	if len(parts) < 4 || !dateRe.MatchString(parts[0]) {
		return Commit{}, false
	}
	commit := Commit{
		Day:     parts[0],
		Month:   parts[0][:7],
		Author:  parts[1],
		Hash:    parts[2],
		Parents: parts[3],
	}
	if len(parts) == 5 {
		commit.Subject = parts[4]
	}
	return commit, true
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
