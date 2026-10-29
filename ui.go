package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
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
	modeCommits
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

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
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

type commitItem struct {
	commit Commit
}

func (c commitItem) Title() string {
	title := c.commit.Subject
	if title == "" {
		title = "(no description set)"
	}
	return title
}

func (c commitItem) Description() string {
	size := WithUnit(c.commit.Size(), "line")
	if parentCount(c.commit) >= 2 {
		size = "merge"
	}
	return fmt.Sprintf("%s · %s", dayLabel(c.commit.Day), size)
}

// dayLabel renders "2024-03-15" as "Mar 15".
func dayLabel(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return t.Format("Jan 2")
}

func (c commitItem) FilterValue() string { return c.commit.Subject }

type model struct {
	months        []MonthSummary
	commits       []Commit
	list          list.Model
	mode          mode
	ready         bool
	height        int
	currentMonth  string
	currentAuthor string
	repo          *Repo
	slug          string
	hasDelta      bool
}

func newModel(commits []Commit, months []MonthSummary, repo *Repo, slug string, hasDelta bool) model {
	items := make([]list.Item, len(months))
	for i, summary := range months {
		items[i] = monthItem{summary: summary}
	}

	l := list.New(items, list.NewDefaultDelegate(), listWidth, 20)
	l.Title = "Months"
	l.SetStatusBarItemName("month", "months")
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return model{commits: commits, months: months, list: l, mode: modeMonths, repo: repo, slug: slug, hasDelta: hasDelta}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.height = msg.Height
		m.list.SetSize(msg.Width/3, msg.Height-1)
		return m, nil

	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.mode {
			case modeAuthors:
				return m.switchToMonths()
			case modeCommits:
				return m.switchToAuthors(m.currentMonth)
			}
			return m, tea.Quit
		case "enter":
			switch m.mode {
			case modeMonths:
				if summary, ok := m.selectedMonth(); ok {
					return m.switchToAuthors(summary.Month)
				}
			case modeAuthors:
				if item, ok := m.list.SelectedItem().(authorItem); ok {
					return m.switchToCommits(item.stats.Author)
				}
			}
		case "o":
			if m.mode == modeCommits && m.slug != "" {
				if item, ok := m.list.SelectedItem().(commitItem); ok {
					url := fmt.Sprintf("https://github.com/%s/commit/%s", m.slug, item.commit.Hash)
					if cmd := openBrowser(url); cmd != nil {
						return m, tea.ExecProcess(cmd, nil)
					}
				}
			}
		case "p":
			if m.mode == modeCommits && m.hasDelta {
				if item, ok := m.list.SelectedItem().(commitItem); ok {
					return m, tea.ExecProcess(m.repo.DiffCommand(item.commit.Hash), nil)
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
	index := 0
	for i, summary := range m.months {
		items[i] = monthItem{summary: summary}
		if summary.Month == m.currentMonth {
			index = i
		}
	}
	m.list.Title = "Months"
	m.list.SetStatusBarItemName("month", "months")
	m.list.SetItems(items)
	m.list.Select(index)
	m.mode = modeMonths
	return m, nil
}

func (m model) switchToAuthors(month string) (tea.Model, tea.Cmd) {
	m.currentMonth = month
	for _, summary := range m.months {
		if summary.Month != month {
			continue
		}
		items := make([]list.Item, len(summary.Authors))
		index := 0
		for i, stats := range summary.Authors {
			items[i] = authorItem{stats: stats, month: month}
			if stats.Author == m.currentAuthor {
				index = i
			}
		}
		m.list.Title = monthLabel(month)
		m.list.SetStatusBarItemName("author", "authors")
		m.list.SetItems(items)
		m.list.Select(index)
	}
	m.mode = modeAuthors
	return m, nil
}

func (m model) switchToCommits(author string) (tea.Model, tea.Cmd) {
	m.currentAuthor = author

	var items []list.Item
	for _, c := range m.commits {
		if c.Month == m.currentMonth && c.Author == author {
			items = append(items, commitItem{commit: c})
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	m.list.Title = author
	m.list.SetStatusBarItemName("commit", "commits")
	m.list.SetItems(items)
	m.list.Select(0)
	m.mode = modeCommits
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

// renderCommitPreview shows the details of one commit.
func renderCommitPreview(item commitItem) string {
	c := item.commit
	var b strings.Builder
	if parentCount(c) >= 2 {
		fmt.Fprintf(&b, "Merge commit\n\n")
	}
	fmt.Fprintf(&b, "%-12s %s\n", "Date", c.Day)
	fmt.Fprintf(&b, "%-12s %s\n", "Author", c.Author)
	fmt.Fprintf(&b, "%-12s %s\n", "Hash", shortHash(c.Hash))
	fmt.Fprintf(&b, "%-12s +%s\n", "Insertions", WithUnit(c.Insertions, "line"))
	fmt.Fprintf(&b, "%-12s -%s\n", "Deletions", WithUnit(c.Deletions, "line"))
	return b.String()
}

func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// openBrowser returns the command to open a URL in the browser, or nil on
// unsupported platforms.
func openBrowser(url string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url)
	case "linux":
		return exec.Command("xdg-open", url)
	default:
		return nil
	}
}

// footer is the actions bar, listing the keys that work in the current view.
func (m model) footer() string {
	keys := []string{"↑/↓ move"}
	switch m.mode {
	case modeMonths:
		keys = append(keys, "enter contributors", "esc quit")
	case modeAuthors:
		keys = append(keys, "enter commits", "esc back", "q quit")
	case modeCommits:
		if m.slug != "" {
			keys = append(keys, "o open on github")
		}
		if m.hasDelta {
			keys = append(keys, "p diff")
		}
		keys = append(keys, "esc back", "q quit")
	}
	return footerStyle.Render(strings.Join(keys, " · "))
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
	case modeCommits:
		if item, ok := m.list.SelectedItem().(commitItem); ok {
			preview = previewHeaderStyle.Render(item.commit.Subject) + "\n" + renderCommitPreview(item)
		} else {
			preview = "No commits this month."
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), m.previewPane(preview)) + "\n" + m.footer()
}

// previewPane pads the preview and clamps it to the terminal height, so a
// month with many contributors cannot push the header off the top of the
// screen.
func (m model) previewPane(preview string) string {
	return lipgloss.NewStyle().MaxHeight(m.height - 1).Render(previewStyle.Render(preview))
}

// RunUI loads the repo's history, filters it, and runs the TUI.
func RunUI(w io.Writer, commits []Commit, months []MonthSummary, repo *Repo, slug string, hasDelta bool) error {
	if len(months) == 0 {
		fmt.Fprintln(w, "No commits found.")
		return nil
	}
	_, err := tea.NewProgram(newModel(commits, months, repo, slug, hasDelta), tea.WithAltScreen()).Run()
	return err
}
