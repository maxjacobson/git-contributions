package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/x/ansi"
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

func TestDayLabelWithYear(t *testing.T) {
	if got := dayLabelWithYear("2025-12-30"); got != "Dec 30, 2025" {
		t.Errorf("got %q, want %q", got, "Dec 30, 2025")
	}
	if got := dayLabelWithYear("nonsense"); got != "nonsense" {
		t.Errorf("got %q, want it passed through", got)
	}
}

func TestBreadcrumbParts(t *testing.T) {
	months := model{
		mode: modeMonths,
		list: list.New([]list.Item{periodItem{summary: PeriodSummary{Period: "2024-03"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := months.breadcrumbParts(); !reflect.DeepEqual(got, []string{"March 2024"}) {
		t.Errorf("months breadcrumb: got %v", got)
	}

	authors := model{
		mode:          modeAuthors,
		currentPeriod: "2024",
		list:          list.New([]list.Item{authorItem{stats: AuthorStats{Author: "Alice"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := authors.breadcrumbParts(); !reflect.DeepEqual(got, []string{"2024", "Alice"}) {
		t.Errorf("authors breadcrumb: got %v", got)
	}

	teamCommits := model{
		mode:          modeCommits,
		currentPeriod: "2024-03",
		currentTeam:   "Payments",
		list:          list.New([]list.Item{commitItem{commit: Commit{Subject: "Fix the thing"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := teamCommits.breadcrumbParts(); !reflect.DeepEqual(got, []string{"March 2024", "Payments", "Fix the thing"}) {
		t.Errorf("team commits breadcrumb: got %v", got)
	}

	withProject := model{
		mode:    modeMonths,
		project: "rubocop",
		list:    list.New([]list.Item{periodItem{summary: PeriodSummary{Period: "2024-03"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if got := withProject.breadcrumbParts(); !reflect.DeepEqual(got, []string{"March 2024"}) {
		t.Errorf("project breadcrumb: got %v", got)
	}
	if got := withProject.headingView(); !strings.Contains(got, "contributions to ") || !strings.Contains(got, "rubocop") || !strings.Contains(got, "by month") {
		t.Errorf("heading: got %q, want it to name the project by month", got)
	}

	yearCommits := model{
		mode:          modeCommits,
		currentPeriod: "2024",
		project:       "rubocop",
	}
	if got := yearCommits.headingView(); !strings.Contains(got, "by year") {
		t.Errorf("heading: got %q, want it to say by year", got)
	}
}

func TestCrumbSpans(t *testing.T) {
	authors := model{
		mode:          modeAuthors,
		currentPeriod: "2024",
		list:          list.New([]list.Item{authorItem{stats: AuthorStats{Author: "Alice"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	spans := authors.crumbSpans()
	if len(spans) != 1 || spans[0] != (crumbSpan{start: 0, end: 4, kind: crumbPeriod}) {
		t.Errorf("authors crumb spans: got %v, want one period span over \"2024\"", spans)
	}
	if kind, ok := authors.crumbAt(2); !ok || kind != crumbPeriod {
		t.Errorf("crumbAt(2): got %d, %v, want crumbPeriod", kind, ok)
	}
	if _, ok := authors.crumbAt(5); ok {
		t.Errorf("crumbAt(5): got a span, want none: \"Alice\" is not clickable")
	}

	teamCommits := model{
		mode:          modeCommits,
		currentPeriod: "2024-03",
		currentTeam:   "Payments",
		list:          list.New([]list.Item{commitItem{commit: Commit{Subject: "Fix the thing"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	spans = teamCommits.crumbSpans()
	if len(spans) != 2 || spans[0] != (crumbSpan{start: 0, end: 10, kind: crumbPeriod}) || spans[1] != (crumbSpan{start: 13, end: 21, kind: crumbTeam}) {
		t.Errorf("team commits crumb spans: got %v, want period then team spans", spans)
	}

	months := model{
		mode: modeMonths,
		list: list.New([]list.Item{periodItem{summary: PeriodSummary{Period: "2024-03"}}}, list.NewDefaultDelegate(), 20, 5),
	}
	if spans := months.crumbSpans(); len(spans) != 0 {
		t.Errorf("months crumb spans: got %v, want none", spans)
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

func TestFillBackgrounds(t *testing.T) {
	input := strings.Join([]string{"ab\x1b[0K", "abcdef\x1b[0K"}, "\n")
	got := strings.Split(fillBackgrounds(input, 4), "\n")
	for i, line := range got {
		if w := ansi.StringWidth(line); w != 6 {
			t.Errorf("line %d: got width %d, want padded to the widest line (6)", i, w)
		}
	}

	got = strings.Split(fillBackgrounds("ab\x1b[0K", 10), "\n")
	if w := ansi.StringWidth(got[0]); w != 10 {
		t.Errorf("floor: got width %d, want padded to minWidth (10)", w)
	}

	if got := naturalLineWidth("ab\x1b[0K ignore"); got != 2 {
		t.Errorf("natural width with erase sequence: got %d, want 2", got)
	}
	if got := naturalLineWidth("plain"); got != 5 {
		t.Errorf("natural width without erase sequence: got %d, want 5", got)
	}
}
