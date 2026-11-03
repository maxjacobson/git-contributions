package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestMonthLabel(t *testing.T) {
	if got := monthLabel("2026-10"); got != "October 2026" {
		t.Errorf("got %q, want %q", got, "October 2026")
	}
	if got := monthLabel("nonsense"); got != "nonsense" {
		t.Errorf("got %q, want it passed through", got)
	}
}

func TestPeriodLabel(t *testing.T) {
	if got := periodLabel("2024"); got != "2024" {
		t.Errorf("got %q, want %q", got, "2024")
	}
}

func TestDayLabel(t *testing.T) {
	if got := dayLabel("2024-03-15"); got != "Mar 15" {
		t.Errorf("got %q, want %q", got, "Mar 15")
	}
	if got := dayLabel("nonsense"); got != "nonsense" {
		t.Errorf("got %q, want it passed through", got)
	}
}

func TestBreadcrumbParts(t *testing.T) {
	months := model{
		mode: modeMonths,
		list: list.New([]list.Item{periodItem{summary: PeriodSummary{Period: "2024-03"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := months.breadcrumbParts(); !reflect.DeepEqual(got, []string{"Months", "March 2024"}) {
		t.Errorf("months breadcrumb: got %v", got)
	}

	authors := model{
		mode:          modeAuthors,
		currentPeriod: "2024",
		list:          list.New([]list.Item{authorItem{stats: AuthorStats{Author: "Alice"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := authors.breadcrumbParts(); !reflect.DeepEqual(got, []string{"Years", "2024", "Alice"}) {
		t.Errorf("authors breadcrumb: got %v", got)
	}

	teamCommits := model{
		mode:          modeCommits,
		currentPeriod: "2024-03",
		currentTeam:   "Payments",
		list:          list.New([]list.Item{commitItem{commit: Commit{Subject: "Fix the thing"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := teamCommits.breadcrumbParts(); !reflect.DeepEqual(got, []string{"Months", "March 2024", "Payments", "Fix the thing"}) {
		t.Errorf("team commits breadcrumb: got %v", got)
	}

	withProject := model{
		mode:    modeMonths,
		project: "rubocop",
		list:    list.New([]list.Item{periodItem{summary: PeriodSummary{Period: "2024-03"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := withProject.breadcrumbParts(); !reflect.DeepEqual(got, []string{"Months", "March 2024"}) {
		t.Errorf("project breadcrumb: got %v", got)
	}
	if got := withProject.headingView(); !strings.Contains(got, "contributions to ") || !strings.Contains(got, "rubocop") {
		t.Errorf("heading: got %q, want it to name the project", got)
	}
}

func TestProjectName(t *testing.T) {
	if got := projectName(&Repo{Dir: "/Users/max/src/github/maxjacobson/rubocop"}); got != "rubocop" {
		t.Errorf("jj repo: got %q, want %q", got, "rubocop")
	}
	if got := projectName(&Repo{Dir: "."}); got != "" {
		t.Errorf("bare dot dir: got %q, want empty", got)
	}
}
