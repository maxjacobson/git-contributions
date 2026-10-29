package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// headerRe matches the log format line for one commit: "2024-03;;;Alice;;;".
var headerRe = regexp.MustCompile(`^(\d{4}-\d{2});;;(.*);;;$`)

var (
	insertionsRe = regexp.MustCompile(`(\d+) insertion`)
	deletionsRe  = regexp.MustCompile(`(\d+) deletion`)
)

// ParseLog parses the output of
// git log --pretty=tformat:"%ad;;;%aN;;;" --date=format:"%Y-%m" --shortstat
// into commits. A commit with no changes has an empty stat section.
func ParseLog(output string) ([]Commit, error) {
	var commits []Commit
	current := Commit{}
	started := false

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if matches := headerRe.FindStringSubmatch(line); matches != nil {
			if started {
				commits = append(commits, current)
			}
			current = Commit{Month: matches[1], Author: matches[2]}
			started = true
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
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if started {
		commits = append(commits, current)
	}
	return commits, nil
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
