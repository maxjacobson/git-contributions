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

	if err := RunUI(os.Stdout, SummarizeMonths(commits)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
