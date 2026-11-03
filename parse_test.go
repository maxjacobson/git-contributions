package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseLog(t *testing.T) {
	output := strings.Join([]string{
		"2024-03-15;;;Alice;;;hash1;;;abc;;;Fix the thing",
		" 3 files changed, 10 insertions(+), 2 deletions(-)",
		"",
		"2024-03-20;;;Bob;;;hash2;;;",
		"",
		"2024-02-01;;;Alice;;;hash3;;;def;;;Add a thing",
		" 1 file changed, 5 insertions(+)",
		"",
		"2024-02-02;;;Alice;;;hash4;;;def;;;Another thing",
		" 1 file changed, 1 deletion(-)",
		"",
		"2024-01-31;;;Carol;;;hash5;;;",
		"",
	}, "\n")

	commits, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}

	want := []Commit{
		{Day: "2024-03-15", Month: "2024-03", Author: "Alice", Hash: "hash1", Parents: "abc", Subject: "Fix the thing", Insertions: 10, Deletions: 2},
		{Day: "2024-02-01", Month: "2024-02", Author: "Alice", Hash: "hash3", Parents: "def", Subject: "Add a thing", Insertions: 5},
		{Day: "2024-02-02", Month: "2024-02", Author: "Alice", Hash: "hash4", Parents: "def", Subject: "Another thing", Deletions: 1},
	}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
}

func TestParseLogKeepsMergeWithoutStat(t *testing.T) {
	output := strings.Join([]string{
		"2024-03-15;;;Alice;;;hash1;;;abc def;;;Merge branch 'x'",
		"",
		"2024-03-16;;;Bob;;;hash2;;;abc def;;;Merge branch 'y'",
		"",
	}, "\n")

	commits, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}

	want := []Commit{
		{Day: "2024-03-15", Month: "2024-03", Author: "Alice", Hash: "hash1", Parents: "abc def", Subject: "Merge branch 'x'"},
		{Day: "2024-03-16", Month: "2024-03", Author: "Bob", Hash: "hash2", Parents: "abc def", Subject: "Merge branch 'y'"},
	}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
}

func TestParseLogSkipsEmptyCommit(t *testing.T) {
	commits, err := ParseLog("2024-03-15;;;Alice;;;hash1;;;abc;;;empty\n")
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("got %v, want no commits", commits)
	}
}

func TestParseLogKeepsZeroLineCommit(t *testing.T) {
	output := strings.Join([]string{
		"2024-03-15;;;Alice;;;hash1;;;abc;;;chmod",
		" 1 file changed",
		"",
	}, "\n")

	commits, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	want := []Commit{{Day: "2024-03-15", Month: "2024-03", Author: "Alice", Hash: "hash1", Parents: "abc", Subject: "chmod"}}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
}

func TestParseLogSubjectWithSeparators(t *testing.T) {
	output := strings.Join([]string{
		"2024-03-15;;;Alice;;;hash1;;;abc;;;weird ;;; subject",
		" 1 file changed, 1 insertion(+)",
		"",
	}, "\n")

	commits, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	want := []Commit{{Day: "2024-03-15", Month: "2024-03", Author: "Alice", Hash: "hash1", Parents: "abc", Subject: "weird ;;; subject", Insertions: 1}}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
}

func TestParseLogAuthorWithSpaces(t *testing.T) {
	output := strings.Join([]string{
		"2024-03-15;;;Dependabot Bot;;;hash1;;;abc;;;Bump foo",
		" 1 file changed, 1 insertion(+)",
		"",
	}, "\n")

	commits, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	want := []Commit{{Day: "2024-03-15", Month: "2024-03", Author: "Dependabot Bot", Hash: "hash1", Parents: "abc", Subject: "Bump foo", Insertions: 1}}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
}

func TestParseLogEmpty(t *testing.T) {
	commits, err := ParseLog("")
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("got %v, want no commits", commits)
	}
}

func TestLogParserFeedingLinesMatchesParseLog(t *testing.T) {
	output := strings.Join([]string{
		"2024-03-15;;;Alice;;;hash1;;;abc;;;Fix the thing",
		" 3 files changed, 10 insertions(+), 2 deletions(-)",
		"",
		"2024-03-20;;;Bob;;;hash2;;;",
		"",
		"2024-02-01;;;Alice;;;hash3;;;def;;;Add a thing",
		" 1 file changed, 5 insertions(+)",
		"",
	}, "\n")

	p := &logParser{}
	for _, line := range strings.Split(output, "\n") {
		p.Feed(line)
	}
	commits, err := p.Finish()
	if err != nil {
		t.Fatalf("Finish returned error: %v", err)
	}

	want, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
	if p.Seen != 3 {
		t.Errorf("got Seen %d, want 3", p.Seen)
	}
	if p.Oldest != "2024-02-01" {
		t.Errorf("got Oldest %q, want 2024-02-01", p.Oldest)
	}
}

func TestLogParserTracksProgressPerLine(t *testing.T) {
	p := &logParser{}
	p.Feed("2024-03-15;;;Alice;;;hash1;;;abc;;;Fix the thing")
	if p.Seen != 1 || p.Oldest != "2024-03-15" {
		t.Fatalf("got Seen %d, Oldest %q, want 1 and 2024-03-15", p.Seen, p.Oldest)
	}
	p.Feed("2024-03-16;;;Alice;;;hash2;;;abc;;;More")
	if p.Seen != 2 || p.Oldest != "2024-03-16" {
		t.Fatalf("got Seen %d, Oldest %q, want 2 and 2024-03-16", p.Seen, p.Oldest)
	}
	commits, err := p.Finish()
	if err != nil {
		t.Fatalf("Finish returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("got %v, want no commits", commits)
	}
}
