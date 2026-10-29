package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseLog(t *testing.T) {
	output := strings.Join([]string{
		"2024-03;;;Alice;;;",
		" 3 files changed, 10 insertions(+), 2 deletions(-)",
		"",
		"2024-03;;;Bob;;;",
		"",
		"2024-02;;;Alice;;;",
		" 1 file changed, 5 insertions(+)",
		"",
		"2024-02;;;Alice;;;",
		" 1 file changed, 1 deletion(-)",
		"",
		"2024-01;;;Carol;;;",
		"",
	}, "\n")

	commits, err := ParseLog(output)
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}

	want := []Commit{
		{Month: "2024-03", Author: "Alice", Insertions: 10, Deletions: 2},
		{Month: "2024-03", Author: "Bob"},
		{Month: "2024-02", Author: "Alice", Insertions: 5},
		{Month: "2024-02", Author: "Alice", Deletions: 1},
		{Month: "2024-01", Author: "Carol"},
	}
	if !reflect.DeepEqual(commits, want) {
		t.Errorf("got %v, want %v", commits, want)
	}
}

func TestParseLogAuthorWithSpaces(t *testing.T) {
	commits, err := ParseLog("2024-03;;;Dependabot Bot;;;\n")
	if err != nil {
		t.Fatalf("ParseLog returned error: %v", err)
	}
	want := []Commit{{Month: "2024-03", Author: "Dependabot Bot"}}
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
