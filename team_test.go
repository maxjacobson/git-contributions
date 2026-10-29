package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTeamFlowEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	os.Chdir(dir)

	content := `{
		"Alice": {"team": "Core", "active": true, "note": "founder"},
		"Bob": {"team": "Core", "active": false},
		"Carol": {"team": "Platform", "active": true}
	}`
	metadataPath := filepath.Join(dir, ".git", "git-contributions-metadata")
	os.WriteFile(metadataPath, []byte(content), 0o644)

	repo := DetectRepo()
	metadata, err := LoadMetadata(repo.MetadataFile())
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	commits := []Commit{
		{Month: "2024-03", Author: "Alice", Insertions: 10, Deletions: 2},
		{Month: "2024-03", Author: "Bob", Insertions: 4, Deletions: 4},
		{Month: "2024-03", Author: "Carol", Insertions: 1},
	}
	teams := SummarizeTeams(commits, func(author string) string {
		profile, _ := metadata.Lookup(author)
		return profile.Team
	})

	if len(teams) != 2 {
		t.Fatalf("got %d teams, want 2", len(teams))
	}
	if teams[0].Team != "Core" || teams[0].Stats.Commits != 2 || len(teams[0].Members) != 2 {
		t.Errorf("Core team wrong: %+v", teams[0])
	}
	if teams[1].Team != "Platform" || teams[1].Members[0].Author != "Carol" {
		t.Errorf("Platform team wrong: %+v", teams[1])
	}

	profile, ok := metadata.Lookup("Bob")
	if !ok || profile.Team != "Core" || profile.Active {
		t.Errorf("Bob profile wrong: %+v %v", profile, ok)
	}
	item := authorItem{stats: AuthorStats{Author: "Bob", Commits: 1}, profile: profile}
	fmt.Printf("description: %q\n", item.Description())
}
