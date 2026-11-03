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

// logParser incrementally parses the output of
// git log --pretty=tformat:"%ad;;;%aN;;;%H;;;%P;;;%s" --date=format:"%Y-%m-%d" --shortstat
// one line at a time, so a streaming reader can report progress. Commits with
// no stat line and fewer than two parents are skipped as empty, since a merge
// commit has no stat but is a real commit, while an empty commit is not a
// contribution. Seen counts every commit header, including skipped ones, so
// it tracks git's own position in the walk; oldest is the day of the most
// recent header, which git emits newest first.
type logParser struct {
	commits []Commit
	current Commit
	started bool
	hasStat bool
	err     error
	Seen    int
	Oldest  string
}

func (p *logParser) flush() {
	if p.started && (p.hasStat || parentCount(p.current) >= 2) {
		p.commits = append(p.commits, p.current)
	}
}

func (p *logParser) Feed(line string) {
	if p.err != nil {
		return
	}
	if parts, ok := parseHeader(line); ok {
		p.flush()
		p.current = parts
		p.started = true
		p.hasStat = false
		p.Seen++
		p.Oldest = parts.Day
		return
	}
	if !p.started {
		if strings.TrimSpace(line) == "" {
			return
		}
		p.err = fmt.Errorf("unexpected log line before first commit: %q", line)
		return
	}
	if strings.TrimSpace(line) == "" {
		return
	}
	p.current.Insertions = countMatches(insertionsRe, line)
	p.current.Deletions = countMatches(deletionsRe, line)
	p.hasStat = true
}

func (p *logParser) Finish() ([]Commit, error) {
	p.flush()
	return p.commits, p.err
}

// ParseLog parses the same format as logParser from a full output string.
func ParseLog(output string) ([]Commit, error) {
	p := &logParser{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		p.Feed(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return p.Finish()
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
