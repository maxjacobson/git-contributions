package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	includeDependabot := flag.Bool("include-dependabot", false, "include commits by dependabot")
	flag.Parse()

	repo := DetectRepo()
	commits, err := repo.LoadCommits()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !*includeDependabot {
		commits = FilterDependabot(commits)
	}

	slug := repo.GitHubSlug()

	metadata, err := LoadMetadata(repo.MetadataFile())
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: ignoring contributor metadata: %v\n", err)
	}

	if err := RunUI(os.Stdout, commits, SummarizeMonths(commits), repo, slug, metadata); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
