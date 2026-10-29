package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Commit is one commit, reduced to what this tool cares about.
type Commit struct {
	Day        string // "2024-03-15"
	Month      string // "2024-03"
	Author     string
	Hash       string
	Subject    string
	Parents    string // space-separated parent hashes, empty for root commits
	Insertions int
	Deletions  int
}

// Size is the total number of lines a commit touched.
func (c Commit) Size() int {
	return c.Insertions + c.Deletions
}

// AuthorStats summarizes one author's contributions in one month.
type AuthorStats struct {
	Author     string
	Commits    int
	MergeCount int
	MedianSize int
	MeanSize   int
	Insertions int
	Deletions  int
}

// PeriodSummary describes one period's contributions: a month ("2024-03")
// or a year ("2024").
type PeriodSummary struct {
	Period       string
	TotalCommits int
	Authors      []AuthorStats // sorted by commit count desc, then name
}

// Sum returns the total of the given numbers.
func Sum(numbers []int) int {
	total := 0
	for _, n := range numbers {
		total += n
	}
	return total
}

// Median returns the median, rounding down like the original script.
func Median(numbers []int) int {
	if len(numbers) == 0 {
		return 0
	}
	sorted := append([]int(nil), numbers...)
	sort.Ints(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// Mean returns the mean, rounding down like the original script.
func Mean(numbers []int) int {
	if len(numbers) == 0 {
		return 0
	}
	return Sum(numbers) / len(numbers)
}

// comma groups a number's digits in thousands, eg 1234567 as "1,234,567".
func comma(n int) string {
	negative := n < 0
	if negative {
		n = -n
	}
	s := strconv.Itoa(n)
	var groups []string
	for len(s) > 3 {
		groups = append([]string{s[len(s)-3:]}, groups...)
		s = s[:len(s)-3]
	}
	groups = append([]string{s}, groups...)
	out := strings.Join(groups, ",")
	if negative {
		out = "-" + out
	}
	return out
}

// WithUnit pluralizes a count with its unit.
func WithUnit(count int, unit string) string {
	if count == 1 {
		return fmt.Sprintf("%s %s", comma(count), unit)
	}
	return fmt.Sprintf("%s %ss", comma(count), unit)
}

// TeamSummary pools one team's contributions in one month.
type TeamSummary struct {
	Team    string
	Stats   AuthorStats   // pooled; Author is the team name
	Members []AuthorStats // per-member counts, busiest first
}

// SummarizeTeams pools commits by the team of their author. Authors whose
// teamOf returns "" land in a "(no team)" bucket. Teams are sorted by commit
// count (descending), then by name.
func SummarizeTeams(commits []Commit, teamOf func(string) string) []TeamSummary {
	byTeam := map[string][]Commit{}
	for _, c := range commits {
		team := teamOf(c.Author)
		if team == "" {
			team = "(no team)"
		}
		byTeam[team] = append(byTeam[team], c)
	}

	names := make([]string, 0, len(byTeam))
	for team := range byTeam {
		names = append(names, team)
	}
	sort.Strings(names)

	teams := make([]TeamSummary, 0, len(names))
	for _, team := range names {
		teamCommits := byTeam[team]
		pooled := make([]Commit, len(teamCommits))
		for i, c := range teamCommits {
			pooled[i] = c
			pooled[i].Author = team
		}
		teams = append(teams, TeamSummary{
			Team:    team,
			Stats:   SummarizeAuthors(pooled)[0],
			Members: SummarizeAuthors(teamCommits),
		})
	}

	sort.SliceStable(teams, func(i, j int) bool {
		if teams[i].Stats.Commits != teams[j].Stats.Commits {
			return teams[i].Stats.Commits > teams[j].Stats.Commits
		}
		return teams[i].Team < teams[j].Team
	})
	return teams
}

// SummarizeMonths groups commits by month, newest first, with each month's
// authors sorted by commit count (descending), then by name.
func SummarizeMonths(commits []Commit) []PeriodSummary {
	return summarizePeriods(commits, func(c Commit) string { return c.Month })
}

// SummarizeYears groups commits by year, newest first, like SummarizeMonths.
func SummarizeYears(commits []Commit) []PeriodSummary {
	return summarizePeriods(commits, func(c Commit) string { return c.Month[:4] })
}

// isYear reports whether a period is a year ("2024") rather than a month
// ("2024-03").
func isYear(period string) bool {
	return len(period) == 4
}

// inPeriod reports whether a commit falls within the period.
func inPeriod(c Commit, period string) bool {
	if isYear(period) {
		return strings.HasPrefix(c.Month, period)
	}
	return c.Month == period
}

func summarizePeriods(commits []Commit, periodOf func(Commit) string) []PeriodSummary {
	byPeriod := map[string][]Commit{}
	for _, c := range commits {
		period := periodOf(c)
		byPeriod[period] = append(byPeriod[period], c)
	}

	periods := make([]string, 0, len(byPeriod))
	for period := range byPeriod {
		periods = append(periods, period)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(periods)))

	summaries := make([]PeriodSummary, 0, len(periods))
	for _, period := range periods {
		summaries = append(summaries, PeriodSummary{
			Period:       period,
			TotalCommits: len(byPeriod[period]),
			Authors:      SummarizeAuthors(byPeriod[period]),
		})
	}
	return summaries
}

// SummarizeAuthors reduces a month's commits to one AuthorStats per author.
func SummarizeAuthors(commits []Commit) []AuthorStats {
	byAuthor := map[string][]Commit{}
	for _, c := range commits {
		byAuthor[c.Author] = append(byAuthor[c.Author], c)
	}

	authors := make([]string, 0, len(byAuthor))
	for author := range byAuthor {
		authors = append(authors, author)
	}
	sort.Strings(authors)

	stats := make([]AuthorStats, 0, len(authors))
	for _, author := range authors {
		commits := byAuthor[author]
		mergeCount := 0
		var sizes, insertions, deletions []int
		for _, c := range commits {
			if parentCount(c) >= 2 {
				mergeCount++
				continue
			}
			sizes = append(sizes, c.Size())
			insertions = append(insertions, c.Insertions)
			deletions = append(deletions, c.Deletions)
		}
		stats = append(stats, AuthorStats{
			Author:     author,
			Commits:    len(commits),
			MergeCount: mergeCount,
			MedianSize: Median(sizes),
			MeanSize:   Mean(sizes),
			Insertions: Sum(insertions),
			Deletions:  Sum(deletions),
		})
	}

	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].Commits != stats[j].Commits {
			return stats[i].Commits > stats[j].Commits
		}
		return strings.ToLower(stats[i].Author) < strings.ToLower(stats[j].Author)
	})
	return stats
}
