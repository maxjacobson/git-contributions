package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listWidth = 32

type mode int

const (
	modeMonths mode = iota
	modeAuthors
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	previewStyle = lipgloss.NewStyle().
			Padding(1, 2)

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				MarginBottom(1)
)

type monthItem struct {
	summary MonthSummary
}

func (m monthItem) Title() string {
	return monthLabel(m.summary.Month)
}

func (m monthItem) Description() string {
	return fmt.Sprintf("%d commits by %d authors", m.summary.TotalCommits, len(m.summary.Authors))
}

func (m monthItem) FilterValue() string { return m.summary.Month }

type authorItem struct {
	stats AuthorStats
	month string
}

func (a authorItem) Title() string {
	return a.stats.Author
}

func (a authorItem) Description() string {
	return fmt.Sprintf("%d commits", a.stats.Commits)
}

func (a authorItem) FilterValue() string { return a.stats.Author }

type model struct {
	months []MonthSummary
	list   list.Model
	mode   mode
	ready  bool
	height int
}

func newModel(months []MonthSummary) model {
	items := make([]list.Item, len(months))
	for i, summary := range months {
		items[i] = monthItem{summary: summary}
	}

	l := list.New(items, list.NewDefaultDelegate(), listWidth, 20)
	l.Title = "Months"
	l.SetStatusBarItemName("month", "months")
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)

	return model{months: months, list: l, mode: modeMonths}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.height = msg.Height
		m.list.SetSize(msg.Width/3, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.mode == modeAuthors {
				return m.switchToMonths()
			}
			return m, tea.Quit
		case "enter":
			if m.mode == modeMonths {
				if summary, ok := m.selectedMonth(); ok {
					return m.switchToAuthors(summary)
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) selectedMonth() (MonthSummary, bool) {
	item, ok := m.list.SelectedItem().(monthItem)
	if !ok {
		return MonthSummary{}, false
	}
	return item.summary, true
}

func (m model) switchToMonths() (tea.Model, tea.Cmd) {
	items := make([]list.Item, len(m.months))
	for i, summary := range m.months {
		items[i] = monthItem{summary: summary}
	}
	m.list.Title = "Months"
	m.list.SetStatusBarItemName("month", "months")
	m.list.SetItems(items)
	m.mode = modeMonths
	return m, nil
}

func (m model) switchToAuthors(summary MonthSummary) (tea.Model, tea.Cmd) {
	items := make([]list.Item, len(summary.Authors))
	for i, stats := range summary.Authors {
		items[i] = authorItem{stats: stats, month: summary.Month}
	}
	m.list.Title = monthLabel(summary.Month)
	m.list.SetStatusBarItemName("author", "authors")
	m.list.SetItems(items)
	m.mode = modeAuthors
	return m, nil
}

func monthLabel(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("January 2006")
}

// renderMonthPreview is the contributor list for a month, busiest first.
func renderMonthPreview(summary MonthSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d commits by %d authors\n\n", summary.TotalCommits, len(summary.Authors))
	for _, stats := range summary.Authors {
		fmt.Fprintf(&b, "%-24s %4d\n", stats.Author, stats.Commits)
	}
	return b.String()
}

// renderAuthorPreview is the per-author metadata for a month, with the same
// columns as the original script.
func renderAuthorPreview(item authorItem) string {
	stats := item.stats
	var b strings.Builder
	fmt.Fprintf(&b, "%-18s %d\n", "Merge commits", stats.MergeCount)
	fmt.Fprintf(&b, "%-18s %d\n", "Non-merge commits", stats.Commits-stats.MergeCount)
	fmt.Fprint(&b, "\n")
	fmt.Fprintf(&b, "%-12s %s\n", "Median size", WithUnit(stats.MedianSize, "line"))
	fmt.Fprintf(&b, "%-12s %s\n", "Mean size", WithUnit(stats.MeanSize, "line"))
	fmt.Fprintf(&b, "%-12s +%s\n", "Insertions", WithUnit(stats.Insertions, "line"))
	fmt.Fprintf(&b, "%-12s -%s\n", "Deletions", WithUnit(stats.Deletions, "line"))
	return b.String()
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var preview string
	switch m.mode {
	case modeMonths:
		if summary, ok := m.selectedMonth(); ok {
			preview = previewHeaderStyle.Render(monthLabel(summary.Month)) + "\n" + renderMonthPreview(summary)
		} else {
			preview = "No months with commits."
		}
	case modeAuthors:
		if item, ok := m.list.SelectedItem().(authorItem); ok {
			preview = previewHeaderStyle.Render(item.stats.Author) + "\n" + renderAuthorPreview(item)
		} else {
			preview = "No contributors this month."
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), m.previewPane(preview))
}

// previewPane pads the preview and clamps it to the terminal height, so a
// month with many contributors cannot push the header off the top of the
// screen.
func (m model) previewPane(preview string) string {
	return lipgloss.NewStyle().MaxHeight(m.height).Render(previewStyle.Render(preview))
}

// RunUI loads the repo's history, filters it, and runs the TUI.
func RunUI(w io.Writer, months []MonthSummary) error {
	if len(months) == 0 {
		fmt.Fprintln(w, "No commits found.")
		return nil
	}
	_, err := tea.NewProgram(newModel(months), tea.WithAltScreen()).Run()
	return err
}
