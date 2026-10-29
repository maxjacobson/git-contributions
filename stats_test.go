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

func TestWithUnit(t *testing.T) {
	if got := WithUnit(1, "line"); got != "1 line" {
		t.Errorf("got %q, want %q", got, "1 line")
	}
	if got := WithUnit(2, "line"); got != "2 lines" {
		t.Errorf("got %q, want %q", got, "2 lines")
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
	if months[0].Month != "2024-03" || months[1].Month != "2024-02" {
		t.Errorf("months not sorted newest first: %v", months)
	}
	if months[0].TotalCommits != 2 {
		t.Errorf("TotalCommits = %d, want 2", months[0].TotalCommits)
	}
	if months[0].Authors[0].Author != "Alice" || months[0].Authors[0].Commits != 1 {
		t.Errorf("tie between Bob and Alice did not break alphabetically: %+v", months[0].Authors)
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
