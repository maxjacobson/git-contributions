package main

import (
	"reflect"
	"testing"
)

func TestMedian(t *testing.T) {
	cases := []struct {
		name    string
		numbers []int
		want    int
	}{
		{"empty", []int{}, 0},
		{"single", []int{7}, 7},
		{"odd", []int{3, 1, 2}, 2},
		{"even", []int{4, 1, 3, 2}, 2},
		{"even rounds down", []int{1, 2}, 1},
		{"unsorted even", []int{10, 1, 8, 3}, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Median(tc.numbers); got != tc.want {
				t.Errorf("Median(%v) = %d, want %d", tc.numbers, got, tc.want)
			}
		})
	}
}

func TestMean(t *testing.T) {
	cases := []struct {
		name    string
		numbers []int
		want    int
	}{
		{"empty", []int{}, 0},
		{"single", []int{7}, 7},
		{"rounds down", []int{1, 2}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Mean(tc.numbers); got != tc.want {
				t.Errorf("Mean(%v) = %d, want %d", tc.numbers, got, tc.want)
			}
		})
	}
}

func TestSum(t *testing.T) {
	if got := Sum([]int{1, 2, 3}); got != 6 {
		t.Errorf("Sum = %d, want 6", got)
	}
	if got := Sum(nil); got != 0 {
		t.Errorf("Sum(nil) = %d, want 0", got)
	}
}

func TestComma(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{-1234, "-1,234"},
	} {
		if got := comma(tc.n); got != tc.want {
			t.Errorf("comma(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestWithUnit(t *testing.T) {
	if got := WithUnit(1, "line"); got != "1 line" {
		t.Errorf("got %q, want %q", got, "1 line")
	}
	if got := WithUnit(2, "line"); got != "2 lines" {
		t.Errorf("got %q, want %q", got, "2 lines")
	}
	if got := WithUnit(23456, "line"); got != "23,456 lines" {
		t.Errorf("got %q, want %q", got, "23,456 lines")
	}
}

func TestSummarizeAuthors(t *testing.T) {
	commits := []Commit{
		{Month: "2024-03", Author: "Alice", Insertions: 10, Deletions: 2},
		{Month: "2024-03", Author: "Bob"},
		{Month: "2024-03", Author: "Alice", Insertions: 4, Deletions: 4},
		{Month: "2024-03", Author: "Alice", Insertions: 1},
		{Month: "2024-03", Author: "Alice", Parents: "abc def"},
	}

	stats := SummarizeAuthors(commits)
	want := []AuthorStats{
		{Author: "Alice", Commits: 4, MergeCount: 1, MedianSize: 8, MeanSize: 7, Insertions: 15, Deletions: 6},
		{Author: "Bob", Commits: 1, MedianSize: 0, MeanSize: 0},
	}
	if !reflect.DeepEqual(stats, want) {
		t.Errorf("got %+v, want %+v", stats, want)
	}
}

func TestSummarizeTeams(t *testing.T) {
	commits := []Commit{
		{Month: "2024-03", Author: "Alice", Insertions: 10, Deletions: 2},
		{Month: "2024-03", Author: "Bob", Insertions: 4, Deletions: 4},
		{Month: "2024-03", Author: "Carol", Insertions: 1},
		{Month: "2024-03", Author: "Alice", Parents: "abc def"},
	}
	teamOf := func(author string) string {
		teams := map[string]string{"Alice": "Core", "Bob": "Core"}
		return teams[author]
	}

	teams := SummarizeTeams(commits, teamOf)
	if len(teams) != 2 {
		t.Fatalf("got %d teams, want 2 (Core and no team)", len(teams))
	}

	if teams[0].Team != "Core" || teams[0].Stats.Commits != 3 {
		t.Errorf("Core should lead with 3 pooled commits: %+v", teams[0].Stats)
	}
	if teams[0].Stats.MedianSize != 10 || teams[0].Stats.MeanSize != 10 || teams[0].Stats.Insertions != 14 {
		t.Errorf("pooled stats wrong: %+v", teams[0].Stats)
	}
	if teams[0].Members[0].Author != "Alice" || teams[0].Members[0].Commits != 2 {
		t.Errorf("roster wrong: %+v", teams[0].Members)
	}
	if teams[1].Team != "(no team)" {
		t.Errorf("unexpected final team: %+v", teams[1].Team)
	}
	if teams[1].Members[0].Author != "Carol" {
		t.Errorf("unassigned Carol should be in the no-team bucket: %+v", teams[1].Members)
	}
}

func TestSummarizeMonths(t *testing.T) {
	commits := []Commit{
		{Month: "2024-02", Author: "Alice"},
		{Month: "2024-03", Author: "Bob"},
		{Month: "2024-03", Author: "Alice", Insertions: 3},
	}

	months := SummarizeMonths(commits)
	if len(months) != 2 {
		t.Fatalf("got %d months, want 2", len(months))
	}
	if months[0].Period != "2024-03" || months[1].Period != "2024-02" {
		t.Errorf("months not sorted newest first: %v", months)
	}
	if months[0].TotalCommits != 2 {
		t.Errorf("TotalCommits = %d, want 2", months[0].TotalCommits)
	}
	if months[0].Authors[0].Author != "Alice" || months[0].Authors[0].Commits != 1 {
		t.Errorf("tie between Bob and Alice did not break alphabetically: %+v", months[0].Authors)
	}
}

func TestSummarizeYears(t *testing.T) {
	commits := []Commit{
		{Month: "2023-11", Author: "Alice"},
		{Month: "2024-01", Author: "Bob"},
		{Month: "2024-06", Author: "Alice", Insertions: 3},
	}

	years := SummarizeYears(commits)
	if len(years) != 2 {
		t.Fatalf("got %d years, want 2", len(years))
	}
	if years[0].Period != "2024" || years[1].Period != "2023" {
		t.Errorf("years not sorted newest first: %v", years)
	}
	if years[0].TotalCommits != 2 {
		t.Errorf("TotalCommits = %d, want 2", years[0].TotalCommits)
	}
	if years[0].Authors[0].Author != "Alice" || years[0].Authors[0].Commits != 1 {
		t.Errorf("tie between Bob and Alice did not break alphabetically: %+v", years[0].Authors)
	}
}

func TestInPeriod(t *testing.T) {
	commit := Commit{Month: "2024-03"}
	if !inPeriod(commit, "2024-03") || !inPeriod(commit, "2024") {
		t.Errorf("commit should match its month and year")
	}
	if inPeriod(commit, "2024-04") || inPeriod(commit, "2023") {
		t.Errorf("commit should not match other periods")
	}
}

func TestFilterDependabot(t *testing.T) {
	commits := []Commit{
		{Month: "2024-03", Author: "Alice"},
		{Month: "2024-03", Author: "dependabot[bot]"},
		{Month: "2024-03", Author: "Bob"},
	}

	kept := FilterDependabot(commits)
	want := []Commit{
		{Month: "2024-03", Author: "Alice"},
		{Month: "2024-03", Author: "Bob"},
	}
	if !reflect.DeepEqual(kept, want) {
		t.Errorf("got %v, want %v", kept, want)
	}
}
