package main

import (
	"os"
	"testing"
)

func TestParseGitHubSlug(t *testing.T) {
	cases := map[string]string{
		"git@github.com:rails/rails.git":             "rails/rails",
		"https://github.com/rails/rails.git":         "rails/rails",
		"https://github.com/rails/rails":             "rails/rails",
		"ssh://git@github.com/maxjacobson/thing.git": "maxjacobson/thing",
		"git@gitlab.com:rails/rails.git":             "",
		"https://gitlab.com/rails/rails.git":         "",
		"https://github.com/rails/rails.wiki.git":    "rails/rails.wiki",
	}
	for url, want := range cases {
		if got := ParseGitHubSlug(url); got != want {
			t.Errorf("ParseGitHubSlug(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestGitHubSlugFromRepo(t *testing.T) {
	dir := os.Getenv("HOME") + "/src/github/rails/rails"
	if _, err := os.Stat(dir); err != nil {
		t.Skip("rails checkout not present")
	}
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	if got := DetectRepo().GitHubSlug(); got != "rails/rails" {
		t.Errorf("got %q, want rails/rails", got)
	}
}
