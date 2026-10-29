package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMetadata(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		metadata, err := LoadMetadata(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := metadata.Lookup("Anyone"); ok {
			t.Error("expected no profiles")
		}
	})

	t.Run("malformed file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "metadata.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadMetadata(path); err == nil {
			t.Error("expected an error for malformed JSON")
		}
	})

	t.Run("lookup by name and alias", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "metadata.json")
		content := `{
			"Akira Matsuda": {"team": "Rails Core", "active": true, "aliases": ["akira"]},
			"Retired Person": {"active": false}
		}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		metadata, err := LoadMetadata(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p, ok := metadata.Lookup("akira"); !ok || p.Team != "Rails Core" || !p.Active {
			t.Errorf("alias lookup failed: %+v %v", p, ok)
		}
		if p, ok := metadata.Lookup("Retired Person"); !ok || p.Active {
			t.Errorf("inactive profile lookup failed: %+v %v", p, ok)
		}
		if _, ok := metadata.Lookup("Nobody"); ok {
			t.Error("unexpected profile for unknown name")
		}
	})
}
