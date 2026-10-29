package main

import (
	"fmt"
	"sort"
	"strings"
)

// Commit is one commit, reduced to what this tool cares about.
type Commit struct {
	Month      string // "2024-03"
	Author     string
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
	Author      string
	Commits     int
	MergeCount  int
	MedianSize  int
	MeanSize    int
	Insertions  int
	Deletions   int
}

// MonthSummary describes one month's contributions.
type MonthSummary struct {
	Month        string
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

// WithUnit pluralizes a count with its unit.
func WithUnit(count int, unit string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, unit)
	}
	return fmt.Sprintf("%d %ss", count, unit)
}

// SummarizeMonths groups commits by month, newest first, with each month's
// authors sorted by commit count (descending), then by name.
func SummarizeMonths(commits []Commit) []MonthSummary {
	byMonth := map[string][]Commit{}
	for _, c := range commits {
		byMonth[c.Month] = append(byMonth[c.Month], c)
	}

	months := make([]string, 0, len(byMonth))
	for month := range byMonth {
		months = append(months, month)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))

	summaries := make([]MonthSummary, 0, len(months))
	for _, month := range months {
		summaries = append(summaries, MonthSummary{
			Month:        month,
			TotalCommits: len(byMonth[month]),
			Authors:      SummarizeAuthors(byMonth[month]),
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
