package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const listWidth = 32

type mode int

const (
	modeMonths mode = iota
	modeAuthors
	modeTeams
	modeCommits
	modeForm
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

	diffPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2)

	formStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 3)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
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
	stats   AuthorStats
	profile Profile
}

func (a authorItem) Title() string {
	return a.stats.Author
}

func (a authorItem) Description() string {
	desc := WithUnit(a.stats.Commits, "commit")
	if a.profile.Team != "" {
		desc += " · " + a.profile.Team
	}
	if a.profile.Note != "" {
		desc += " · " + a.profile.Note
	}
	if a.profile.Active {
		desc += " · active"
	}
	return desc
}

func (a authorItem) FilterValue() string { return a.stats.Author }

type commitItem struct {
	commit     Commit
	showAuthor bool
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
	if c.showAuthor {
		return fmt.Sprintf("%s · %s · %s", c.commit.Author, dayLabel(c.commit.Day), size)
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

type form struct {
	author   string
	inputs   []textinput.Model
	focus    int
	errorMsg string
}

type teamItem struct {
	summary TeamSummary
}

func (t teamItem) Title() string {
	return t.summary.Team
}

func (t teamItem) Description() string {
	return fmt.Sprintf("%s by %d members", WithUnit(t.summary.Stats.Commits, "commit"), len(t.summary.Members))
}

func (t teamItem) FilterValue() string { return t.summary.Team }

func (c commitItem) FilterValue() string { return c.commit.Subject }

type model struct {
	months        []MonthSummary
	commits       []Commit
	list          list.Model
	diffViewport  viewport.Model
	diffWrapped   string
	diffHash      string
	showDiff      bool
	mode          mode
	ready         bool
	height        int
	width         int
	currentMonth  string
	currentAuthor string
	currentTeam   string
	form          form
	repo          *Repo
	slug          string
	metadata      Metadata
}

func newModel(commits []Commit, months []MonthSummary, repo *Repo, slug string, metadata Metadata) model {
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

	return model{commits: commits, months: months, list: l, mode: modeMonths, repo: repo, slug: slug, metadata: metadata}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.height = msg.Height
		m.width = msg.Width
		m.list.SetSize(msg.Width/3, msg.Height-1)
		m.resizeDiff()
		if m.mode == modeCommits && m.showDiff && m.diffHash != "" {
			m.loadDiff(m.diffHash)
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeForm {
			return m.updateForm(msg)
		}
		switch key := msg.String(); key {
		case "ctrl+c":
			return m, tea.Quit
		case "e":
			if m.mode == modeAuthors {
				if item, ok := m.list.SelectedItem().(authorItem); ok {
					return m.openForm(item.stats.Author)
				}
			}
		case "esc", "q":
			switch m.mode {
			case modeAuthors:
				return m.switchToMonths()
			case modeCommits:
				if m.showDiff {
					m.showDiff = false
					return m, nil
				}
				if m.currentTeam != "" {
					return m.switchToTeams(m.currentMonth)
				}
				return m.switchToAuthors(m.currentMonth)
			}
			return m, tea.Quit
		case "t":
			switch m.mode {
			case modeAuthors:
				return m.switchToTeams(m.currentMonth)
			case modeTeams:
				return m.switchToAuthors(m.currentMonth)
			}
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
			case modeTeams:
				if item, ok := m.list.SelectedItem().(teamItem); ok {
					return m.switchToTeamCommits(item.summary.Team)
				}
			case modeCommits:
				if item, ok := m.list.SelectedItem().(commitItem); ok {
					if m.showDiff {
						m.showDiff = false
					} else {
						m.showDiff = true
						if m.diffHash != item.commit.Hash {
							m.loadDiff(item.commit.Hash)
						}
						m.diffViewport.GotoTop()
					}
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
		}

		if m.mode == modeCommits && m.showDiff {
			switch msg.String() {
			case "j", "k", "up", "down", " ", "pgup", "pgdown", "ctrl+d", "ctrl+u", "home", "end":
				var cmd tea.Cmd
				m.diffViewport, cmd = m.diffViewport.Update(msg)
				return m, cmd
			}
		}
	}

	var cmd tea.Cmd
	previousHash := m.selectedCommitHash()
	m.list, cmd = m.list.Update(msg)
	if m.mode == modeCommits && m.showDiff {
		if hash := m.selectedCommitHash(); hash != previousHash && hash != "" {
			m.loadDiff(hash)
		}
	}
	return m, cmd
}

// selectedCommitHash returns the hash of the selected commit in the commit
// view, or an empty string elsewhere.
func (m model) selectedCommitHash() string {
	item, ok := m.list.SelectedItem().(commitItem)
	if !ok {
		return ""
	}
	return item.commit.Hash
}

// diffPaneWidth is the width available for diff text inside the preview,
// leaving room for the pane border and padding.
func (m model) diffPaneWidth() int {
	return m.width - m.width/3 - 6
}

// diffPaneHeight is the height available for the diff viewport, leaving room
// for the footer, the pane border and padding, and the commit subject header.
func (m model) diffPaneHeight() int {
	h := m.height - 7
	if h < 1 {
		h = 1
	}
	return h
}

// resizeDiff refits the viewport and re-wraps the captured diff at the new
// width.
func (m *model) resizeDiff() {
	m.diffViewport.Width = m.diffPaneWidth()
	m.diffViewport.Height = m.diffPaneHeight()
	m.diffViewport.SetContent(ansi.Hardwrap(m.diffWrapped, m.diffPaneWidth(), true))
}

// loadDiff captures the colorized output of showing one commit and puts it
// in the viewport.
func (m *model) loadDiff(hash string) {
	m.diffHash = hash
	m.diffWrapped = ""
	if hash == "" {
		m.resizeDiff()
		return
	}
	output, err := m.repo.DiffOutput(hash, m.diffPaneWidth())
	if err != nil {
		m.diffWrapped = "Could not load diff: " + err.Error()
	} else if strings.TrimSpace(output) == "" {
		m.diffWrapped = "No diff shown for merge commits."
	} else {
		m.diffWrapped = fillBackgrounds(strings.TrimSuffix(output, "\n"), m.diffPaneWidth())
	}
	m.resizeDiff()
	m.diffViewport.GotoTop()
}

// fillBackgrounds expands delta's erase-to-end-of-line sequences into literal
// spaces, since the TUI renderer does not honor them: without this, colored
// line backgrounds stop at the last character instead of running edge to
// edge.
func fillBackgrounds(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = fillLineBackground(line, width)
	}
	return strings.Join(lines, "\n")
}

func fillLineBackground(line string, width int) string {
	for _, seq := range []string{"\x1b[0K", "\x1b[K"} {
		idx := strings.Index(line, seq)
		if idx == -1 {
			continue
		}
		prefix, suffix := line[:idx], line[idx+len(seq):]
		if pad := width - ansi.StringWidth(prefix); pad > 0 {
			return prefix + strings.Repeat(" ", pad) + suffix
		}
		return prefix + suffix
	}
	return line
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
	m.currentTeam = ""
	for _, summary := range m.months {
		if summary.Month != month {
			continue
		}
		items := make([]list.Item, len(summary.Authors))
		index := 0
		for i, stats := range summary.Authors {
			profile, _ := m.metadata.Lookup(stats.Author)
			items[i] = authorItem{stats: stats, profile: profile}
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

// teamOf returns the team of a contributor per the metadata, or an empty
// string when unassigned.
func (m model) teamOf(author string) string {
	profile, _ := m.metadata.Lookup(author)
	return profile.Team
}

func (m model) switchToTeams(month string) (tea.Model, tea.Cmd) {
	m.currentMonth = month
	m.currentTeam = ""

	var monthCommits []Commit
	for _, c := range m.commits {
		if c.Month == month {
			monthCommits = append(monthCommits, c)
		}
	}
	teams := SummarizeTeams(monthCommits, m.teamOf)
	items := make([]list.Item, len(teams))
	for i, summary := range teams {
		items[i] = teamItem{summary: summary}
	}
	m.list.Title = monthLabel(month)
	m.list.SetStatusBarItemName("team", "teams")
	m.list.SetItems(items)
	m.list.Select(0)
	m.mode = modeTeams
	return m, nil
}

func (m model) switchToTeamCommits(team string) (tea.Model, tea.Cmd) {
	m.currentTeam = team

	var items []list.Item
	for _, c := range m.commits {
		inTeam := m.teamOf(c.Author) == team || (team == "(no team)" && m.teamOf(c.Author) == "")
		if c.Month == m.currentMonth && inTeam {
			items = append(items, commitItem{commit: c, showAuthor: true})
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	m.list.Title = team
	m.list.SetStatusBarItemName("commit", "commits")
	m.list.SetItems(items)
	m.list.Select(0)
	m.mode = modeCommits
	return m, nil
}

// openForm opens the profile form for one contributor, prefilled from any
// existing metadata.
func (m model) openForm(author string) (tea.Model, tea.Cmd) {
	profile, _ := m.metadata.Lookup(author)

	note := textinput.New()
	note.Placeholder = "note to self"
	note.Width = 40
	note.SetValue(profile.Note)

	team := textinput.New()
	team.Placeholder = "e.g. Checkout"
	team.Width = 40
	team.SetValue(profile.Team)

	active := textinput.New()
	active.Width = 40
	if profile.Active {
		active.SetValue("y")
	} else {
		active.SetValue("n")
	}

	m.form = form{author: author, inputs: []textinput.Model{note, team, active}}
	cmd := m.form.inputs[0].Focus()
	m.mode = modeForm
	return m, cmd
}

// updateForm routes keys inside the profile form.
func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m.switchToAuthors(m.currentMonth)
		case "enter":
			return m.saveForm()
		case "tab", "down":
			m.form.focus = (m.form.focus + 1) % len(m.form.inputs)
			m.form.inputs[m.form.focus].Focus()
			return m, nil
		case "shift+tab", "up":
			m.form.focus = (m.form.focus + len(m.form.inputs) - 1) % len(m.form.inputs)
			m.form.inputs[m.form.focus].Focus()
			return m, nil
		case " ":
			if m.form.focus == 2 {
				active := &m.form.inputs[2]
				if strings.TrimSpace(active.Value()) == "y" {
					active.SetValue("n")
				} else {
					active.SetValue("y")
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	for i := range m.form.inputs {
		if i == m.form.focus {
			m.form.inputs[i], cmd = m.form.inputs[i].Update(msg)
		} else {
			m.form.inputs[i].Blur()
		}
	}
	return m, cmd
}

// saveForm writes the form's profile to the repo-local metadata file.
func (m model) saveForm() (tea.Model, tea.Cmd) {
	previous, _ := m.metadata.Lookup(m.form.author)
	profile := Profile{
		Note:    strings.TrimSpace(m.form.inputs[0].Value()),
		Team:    strings.TrimSpace(m.form.inputs[1].Value()),
		Active:  parseActive(m.form.inputs[2].Value()),
		Aliases: previous.Aliases,
	}

	path := m.repo.MetadataFile()
	if path == "" {
		m.form.errorMsg = "could not determine the metadata file location"
		return m, nil
	}
	if m.metadata == nil {
		m.metadata = Metadata{}
	}
	m.metadata[m.form.author] = profile
	if err := m.metadata.Save(path); err != nil {
		m.form.errorMsg = err.Error()
		return m, nil
	}
	return m.switchToAuthors(m.currentMonth)
}

// parseActive reads the Active field's y/n-style value.
func parseActive(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "1":
		return true
	}
	return false
}

// formView renders the profile form, centered.
func (m model) formView() string {
	var b strings.Builder
	b.WriteString(previewHeaderStyle.Render(m.form.author) + "\n\n")
	labels := []string{"Note", "Team", "Active"}
	for i, input := range m.form.inputs {
		cursor := "  "
		if i == m.form.focus {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%-7s %s\n", cursor, labels[i]+":", input.View())
	}
	if m.form.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.form.errorMsg) + "\n")
	}
	b.WriteString("\n" + footerStyle.Render("enter save · esc cancel · tab next field · space toggle active"))
	box := formStyle.Render(strings.TrimSuffix(b.String(), "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
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
	m.showDiff = false
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
	var b strings.Builder
	if item.profile.Team != "" || item.profile.Note != "" || item.profile.Active {
		if item.profile.Team != "" {
			fmt.Fprintf(&b, "%-18s %s\n", "Team", item.profile.Team)
		}
		if item.profile.Active {
			fmt.Fprintf(&b, "%-18s %s\n", "Active", "yes")
		}
		if item.profile.Note != "" {
			fmt.Fprintf(&b, "%-18s %s\n", "Note", item.profile.Note)
		}
		fmt.Fprint(&b, "\n")
	}
	b.WriteString(renderStatsTable(item.stats))
	return b.String()
}

// renderStatsTable is the commit-count and size columns shared by the author
// and team previews.
func renderStatsTable(stats AuthorStats) string {
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

// renderTeamPreview is a team's roster followed by its pooled stats.
func renderTeamPreview(item teamItem) string {
	var b strings.Builder
	for _, member := range item.summary.Members {
		fmt.Fprintf(&b, "%-24s %4d\n", member.Author, member.Commits)
	}
	b.WriteString("\n")
	b.WriteString(renderStatsTable(item.summary.Stats))
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
		keys = append(keys, "enter contributors", "esc/q quit")
	case modeAuthors:
		keys = append(keys, "enter commits", "t teams", "esc/q back", "ctrl+c quit")
	case modeTeams:
		keys = append(keys, "enter commits", "t people", "esc/q back", "ctrl+c quit")
	case modeCommits:
		if m.slug != "" {
			keys = append(keys, "o open on github")
		}
		if m.showDiff {
			keys = append(keys, "↑/↓ scroll", "enter hide diff", "esc/q back", "ctrl+c quit")
		} else {
			keys = append(keys, "enter show diff", "esc/q back", "ctrl+c quit")
		}
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
	case modeTeams:
		if item, ok := m.list.SelectedItem().(teamItem); ok {
			preview = previewHeaderStyle.Render(item.summary.Team) + "\n" + renderTeamPreview(item)
		} else {
			preview = "No teams with commits this month."
		}
	case modeCommits:
		if item, ok := m.list.SelectedItem().(commitItem); ok {
			header := previewHeaderStyle.Render(item.commit.Subject)
			if m.showDiff {
				preview = diffPaneStyle.Render(header + "\n" + m.diffViewport.View())
			} else {
				preview = previewStyle.Render(header + "\n" + renderCommitPreview(item))
			}
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
func RunUI(w io.Writer, commits []Commit, months []MonthSummary, repo *Repo, slug string, metadata Metadata) error {
	if len(months) == 0 {
		fmt.Fprintln(w, "No commits found.")
		return nil
	}
	_, err := tea.NewProgram(newModel(commits, months, repo, slug, metadata), tea.WithAltScreen()).Run()
	return err
}
