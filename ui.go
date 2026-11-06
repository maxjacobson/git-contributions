package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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
	modeYears
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

	crumbStyle = footerStyle.Underline(true)

	formStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 3)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

type periodItem struct {
	summary PeriodSummary
}

func (p periodItem) Title() string {
	return periodLabel(p.summary.Period)
}

func (p periodItem) Description() string {
	return fmt.Sprintf("%s commits by %d authors", comma(p.summary.TotalCommits), len(p.summary.Authors))
}

func (p periodItem) FilterValue() string { return p.summary.Period }

type authorItem struct {
	stats   AuthorStats
	profile Profile
	found   bool
}

func (a authorItem) Title() string {
	return a.stats.Author
}

func (a authorItem) Description() string {
	desc := WithUnit(a.stats.Commits, "commit")
	if a.profile.Team != "" {
		desc += " · " + a.profile.Team
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

func dayLabelWithYear(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return t.Format("Jan 2, 2006")
}

type form struct {
	author          string
	inputs          []textinput.Model
	focus           int
	active          bool
	knownTeams      []string
	suggestionIndex int
	errorMsg        string
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
	months        []PeriodSummary
	years         []PeriodSummary
	commits       []Commit
	list          list.Model
	diffViewport  viewport.Model
	diffOutput    string
	diffHash      string
	showDiff      bool
	mode          mode
	ready         bool
	height        int
	width         int
	currentPeriod string
	currentAuthor string
	currentTeam   string
	form          form
	repo          *Repo
	slug          string
	metadata      Metadata
	project       string
}

func newModel(commits []Commit, months []PeriodSummary, repo *Repo, slug string, metadata Metadata) model {
	items := make([]list.Item, len(months))
	for i, summary := range months {
		items[i] = periodItem{summary: summary}
	}

	l := list.New(items, list.NewDefaultDelegate(), listWidth, 20)
	l.SetStatusBarItemName("month", "months")
	l.SetShowStatusBar(true)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return model{commits: commits, months: months, years: SummarizeYears(commits), list: l, mode: modeMonths, repo: repo, slug: slug, metadata: metadata, project: projectName(repo)}
}

// projectName names the repo after the directory it lives in: the jj
// workspace root when there is one, else the current directory.
func projectName(repo *Repo) string {
	dir := repo.Dir
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	name := filepath.Base(dir)
	if name == "." || name == "/" {
		return ""
	}
	return name
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
		m.list.SetSize(msg.Width/3, msg.Height-3)
		m.resizeDiff()
		if m.mode == modeCommits && m.showDiff && m.diffHash != "" {
			m.loadDiff(m.diffHash)
		}
		return m, nil

	case tea.MouseMsg:
		if m.mode == modeForm {
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if msg.Y == breadcrumbRow {
				if kind, ok := m.crumbAt(msg.X); ok {
					switch kind {
					case crumbPeriod:
						return m.switchToPeriod(m.currentPeriod)
					case crumbAuthor:
						return m.switchToAuthors(m.currentPeriod)
					case crumbTeam:
						return m.switchToTeams(m.currentPeriod)
					}
				}
				return m, nil
			}
			if msg.X < listWidth {
				idx, ok := m.listIndexAt(msg.Y)
				if !ok {
					return m, nil
				}
				m.list.Select(idx)
				if m.mode == modeCommits {
					if item, ok := m.list.SelectedItem().(commitItem); ok {
						if m.showDiff && m.diffHash == item.commit.Hash {
							m.showDiff = false
							return m, nil
						}
						m.showDiff = true
						if m.diffHash != item.commit.Hash {
							m.loadDiff(item.commit.Hash)
						} else {
							m.diffViewport.GotoTop()
						}
					}
					return m, nil
				}
				return m.drillIn()
			}
		}
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			if m.mode == modeCommits && m.showDiff {
				var cmd tea.Cmd
				m.diffViewport, cmd = m.diffViewport.Update(msg)
				return m, cmd
			}
			previousHash := m.selectedCommitHash()
			key := tea.KeyMsg{Type: tea.KeyDown}
			if msg.Button == tea.MouseButtonWheelUp {
				key = tea.KeyMsg{Type: tea.KeyUp}
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(key)
			if m.mode == modeCommits && m.showDiff {
				if hash := m.selectedCommitHash(); hash != previousHash && hash != "" {
					m.loadDiff(hash)
				}
			}
			return m, cmd
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
				return m.switchToPeriod(m.currentPeriod)
			case modeTeams:
				return m.switchToAuthors(m.currentPeriod)
			case modeCommits:
				if m.showDiff {
					m.showDiff = false
					return m, nil
				}
				if m.currentTeam != "" {
					return m.switchToTeams(m.currentPeriod)
				}
				return m.switchToAuthors(m.currentPeriod)
			}
			return m, tea.Quit
		case "t":
			switch m.mode {
			case modeAuthors:
				return m.switchToTeams(m.currentPeriod)
			case modeTeams:
				return m.switchToAuthors(m.currentPeriod)
			}
		case "y":
			switch m.mode {
			case modeMonths, modeAuthors, modeTeams:
				return m.switchToYears()
			}
		case "m":
			switch m.mode {
			case modeYears, modeAuthors, modeTeams:
				return m.switchToMonths()
			}
		case "enter":
			if m.mode != modeCommits {
				return m.drillIn()
			}
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
			case "j", "k", "up", "down", "h", "l", "left", "right", " ", "pgup", "pgdown", "ctrl+d", "ctrl+u", "home", "end":
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

func (m model) drillIn() (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeMonths, modeYears:
		if summary, ok := m.selectedPeriod(); ok {
			return m.switchToAuthors(summary.Period)
		}
	case modeAuthors:
		if item, ok := m.list.SelectedItem().(authorItem); ok {
			return m.switchToCommits(item.stats.Author)
		}
	case modeTeams:
		if item, ok := m.list.SelectedItem().(teamItem); ok {
			return m.switchToTeamCommits(item.summary.Team)
		}
	}
	return m, nil
}

const listItemsTop = 3

const listItemRows = 3

func (m model) listIndexAt(y int) (int, bool) {
	row := y - listItemsTop
	if row < 0 {
		return 0, false
	}
	idx := m.list.Paginator.Page*m.list.Paginator.PerPage + row/listItemRows
	if idx >= len(m.list.VisibleItems()) {
		return 0, false
	}
	return idx, true
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
// for the heading, the breadcrumb bar, the footer, the pane border and
// padding.
func (m model) diffPaneHeight() int {
	h := m.height - 7
	if h < 1 {
		h = 1
	}
	return h
}

// resizeDiff refits the viewport to the new size. The captured diff is kept
// unwrapped so long lines can be scrolled into view horizontally.
func (m *model) resizeDiff() {
	m.diffViewport.Width = m.diffPaneWidth()
	m.diffViewport.Height = m.diffPaneHeight()
	m.diffViewport.SetHorizontalStep(1)
	m.diffViewport.SetContent(m.diffOutput)
}

// loadDiff captures the colorized output of showing one commit and puts it
// in the viewport.
func (m *model) loadDiff(hash string) {
	m.diffHash = hash
	m.diffOutput = ""
	if hash == "" {
		m.resizeDiff()
		return
	}
	output, err := m.repo.DiffOutput(hash, diffCaptureWidth())
	if err != nil {
		m.diffOutput = "Could not load diff: " + err.Error()
	} else if strings.TrimSpace(output) == "" {
		m.diffOutput = "No diff shown for merge commits."
	} else {
		m.diffOutput = fillBackgrounds(strings.TrimSuffix(output, "\n"), m.diffPaneWidth())
	}
	m.resizeDiff()
	m.diffViewport.GotoTop()
}

// diffCaptureWidth is the width delta is asked to lay out at: wider than the
// pane so delta does not hard wrap lines, since the viewport scrolls
// horizontally instead, and wide enough that real diff lines stay unwrapped.
func diffCaptureWidth() int {
	return 500
}

// fillBackgrounds expands delta's erase-to-end-of-line sequences into literal
// spaces, since the TUI renderer does not honor them: without this, colored
// line backgrounds stop at the last character instead of running edge to
// edge. Lines are padded to the widest natural line in the output, never
// narrower than minWidth, so scrolling right stops at the real content
// instead of wading through padding.
func fillBackgrounds(s string, minWidth int) string {
	lines := strings.Split(s, "\n")
	width := minWidth
	for _, line := range lines {
		if w := naturalLineWidth(line); w > width {
			width = w
		}
	}
	for i, line := range lines {
		lines[i] = fillLineBackground(line, width)
	}
	return strings.Join(lines, "\n")
}

// naturalLineWidth is the visible width of a line, ignoring the padding that
// follows an erase-to-end-of-line sequence.
func naturalLineWidth(line string) int {
	for _, seq := range []string{"\x1b[0K", "\x1b[K"} {
		if idx := strings.Index(line, seq); idx != -1 {
			return ansi.StringWidth(line[:idx])
		}
	}
	return ansi.StringWidth(line)
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

func (m model) selectedPeriod() (PeriodSummary, bool) {
	item, ok := m.list.SelectedItem().(periodItem)
	if !ok {
		return PeriodSummary{}, false
	}
	return item.summary, true
}

func (m model) switchToMonths() (tea.Model, tea.Cmd) {
	return m.switchToPeriodList(m.months, modeMonths, "month", "months")
}

func (m model) switchToYears() (tea.Model, tea.Cmd) {
	return m.switchToPeriodList(m.years, modeYears, "year", "years")
}

func (m model) switchToPeriodList(summaries []PeriodSummary, target mode, singular, plural string) (tea.Model, tea.Cmd) {
	items := make([]list.Item, len(summaries))
	index := 0
	for i, summary := range summaries {
		items[i] = periodItem{summary: summary}
		if summary.Period == m.currentPeriod {
			index = i
		}
	}
	m.list.SetStatusBarItemName(singular, plural)
	m.list.SetItems(items)
	m.list.Select(index)
	m.mode = target
	return m, nil
}

func (m model) switchToPeriod(period string) (tea.Model, tea.Cmd) {
	if isYear(period) {
		return m.switchToYears()
	}
	return m.switchToMonths()
}

func (m model) switchToAuthors(period string) (tea.Model, tea.Cmd) {
	m.currentPeriod = period
	m.currentTeam = ""
	summaries := m.months
	if isYear(period) {
		summaries = m.years
	}
	for _, summary := range summaries {
		if summary.Period != period {
			continue
		}
		items := make([]list.Item, len(summary.Authors))
		index := 0
		for i, stats := range summary.Authors {
			profile, found := m.metadata.Lookup(stats.Author)
			items[i] = authorItem{stats: stats, profile: profile, found: found}
			if stats.Author == m.currentAuthor {
				index = i
			}
		}
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

func (m model) switchToTeams(period string) (tea.Model, tea.Cmd) {
	m.currentPeriod = period
	m.currentTeam = ""

	var periodCommits []Commit
	for _, c := range m.commits {
		if inPeriod(c, period) {
			periodCommits = append(periodCommits, c)
		}
	}
	teams := SummarizeTeams(periodCommits, m.teamOf)
	items := make([]list.Item, len(teams))
	for i, summary := range teams {
		items[i] = teamItem{summary: summary}
	}
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
		if inPeriod(c, m.currentPeriod) && inTeam {
			items = append(items, commitItem{commit: c, showAuthor: true})
		}
	}
	if items == nil {
		items = []list.Item{}
	}
	m.list.SetStatusBarItemName("commit", "commits")
	m.list.SetItems(items)
	m.list.Select(0)
	m.mode = modeCommits
	return m, nil
}

// breadcrumbParts describes the path to the current view, ending with the
// current selection.
func (m model) breadcrumbParts() []string {
	var parts []string
	switch m.mode {
	case modeMonths, modeYears:
		if summary, ok := m.selectedPeriod(); ok {
			parts = append(parts, periodLabel(summary.Period))
		}
	case modeAuthors:
		parts = append(parts, periodLabel(m.currentPeriod))
		if item, ok := m.list.SelectedItem().(authorItem); ok {
			parts = append(parts, item.stats.Author)
		}
	case modeTeams:
		parts = append(parts, periodLabel(m.currentPeriod))
		if item, ok := m.list.SelectedItem().(teamItem); ok {
			parts = append(parts, item.summary.Team)
		}
	case modeCommits:
		leaf := m.currentAuthor
		if m.currentTeam != "" {
			leaf = m.currentTeam
		}
		parts = append(parts, periodLabel(m.currentPeriod), leaf)
		if item, ok := m.list.SelectedItem().(commitItem); ok {
			parts = append(parts, ansi.Truncate(item.Title(), 60, "…"))
		}
	}
	return parts
}

// headingView names the project above the breadcrumb bar.
func (m model) headingView() string {
	if m.project == "" {
		return ""
	}
	unit := "month"
	if m.mode == modeYears || (m.mode != modeMonths && isYear(m.currentPeriod)) {
		unit = "year"
	}
	return footerStyle.Render("contributions to ") + titleStyle.Render(m.project) + footerStyle.Render(" by "+unit)
}

const breadcrumbRow = 1

const (
	crumbPeriod = iota
	crumbAuthor
	crumbTeam
)

type crumbSpan struct {
	start, end int
	kind       int
}

func (m model) crumbSpans() []crumbSpan {
	parts := m.breadcrumbParts()
	var spans []crumbSpan
	x := 0
	for i, part := range parts {
		w := lipgloss.Width(part)
		kind := -1
		switch m.mode {
		case modeAuthors, modeTeams:
			if i == 0 {
				kind = crumbPeriod
			}
		case modeCommits:
			if i == 0 {
				kind = crumbPeriod
			}
			if i == 1 {
				kind = crumbAuthor
				if m.currentTeam != "" {
					kind = crumbTeam
				}
			}
		}
		if kind >= 0 {
			spans = append(spans, crumbSpan{x, x + w, kind})
		}
		x += w + lipgloss.Width(" ❯ ")
	}
	return spans
}

func (m model) crumbAt(x int) (int, bool) {
	for _, span := range m.crumbSpans() {
		if x >= span.start && x < span.end {
			return span.kind, true
		}
	}
	return 0, false
}

// breadcrumbBar renders the breadcrumb across the top of the screen.
func (m model) breadcrumbBar() string {
	parts := m.breadcrumbParts()
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteString(footerStyle.Render(" ❯ "))
		}
		if i == len(parts)-1 {
			b.WriteString(titleStyle.Render(part))
		} else {
			b.WriteString(crumbStyle.Render(part))
		}
	}
	return b.String()
}

// openForm opens the profile form for one contributor, prefilled from any
// existing metadata.
func (m model) openForm(author string) (tea.Model, tea.Cmd) {
	m.currentAuthor = author
	profile, found := m.metadata.Lookup(author)

	team := textinput.New()
	team.Placeholder = "e.g. Checkout"
	team.Width = 40
	team.SetValue(profile.Team)

	m.form = form{author: author, inputs: []textinput.Model{team}, active: profile.Active || !found, knownTeams: m.knownTeams(), suggestionIndex: -1}
	cmd := m.form.inputs[0].Focus()
	m.mode = modeForm
	return m, cmd
}

// knownTeams collects the distinct teams already present in the metadata,
// sorted.
func (m model) knownTeams() []string {
	set := map[string]bool{}
	for _, profile := range m.metadata {
		if profile.Team != "" {
			set[profile.Team] = true
		}
	}
	teams := make([]string, 0, len(set))
	for team := range set {
		teams = append(teams, team)
	}
	sort.Strings(teams)
	return teams
}

// updateForm routes keys inside the profile form.
func (m model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m.switchToAuthors(m.currentPeriod)
		case "enter":
			return m.saveForm()
		case "tab":
			m.form.focus = (m.form.focus + 1) % 2
			return m, m.focusFormField()
		case "down":
			if m.form.focus == 0 && len(m.form.knownTeams) > 0 {
				if m.form.suggestionIndex < len(m.form.knownTeams) {
					m.form.suggestionIndex++
				}
				m.applySuggestion()
				return m, nil
			}
			m.form.focus = (m.form.focus + 1) % 2
			return m, m.focusFormField()
		case "shift+tab":
			m.form.focus = (m.form.focus + 1) % 2
			return m, m.focusFormField()
		case "up":
			if m.form.focus == 0 && m.form.suggestionIndex >= 0 {
				m.form.suggestionIndex--
				m.applySuggestion()
				return m, nil
			}
			m.form.focus = (m.form.focus + 1) % 2
			return m, m.focusFormField()
		case " ", "left", "right", "h", "l":
			if m.form.focus == 1 {
				m.form.active = !m.form.active
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	for i := range m.form.inputs {
		if i == m.form.focus {
			m.form.inputs[i], cmd = m.form.inputs[i].Update(msg)
			if i == 0 {
				m.form.suggestionIndex = -1
			}
		} else {
			m.form.inputs[i].Blur()
		}
	}
	return m, cmd
}

// applySuggestion fills the team input with the highlighted suggestion,
// unless the highlight is on the "new team" row.
func (m *model) applySuggestion() {
	if m.form.suggestionIndex < 0 || m.form.suggestionIndex >= len(m.form.knownTeams) {
		return
	}
	m.form.inputs[0].SetValue(m.form.knownTeams[m.form.suggestionIndex])
}

// focusFormField focuses the input for the current form focus, blurring the
// others.
func (m *model) focusFormField() tea.Cmd {
	var cmd tea.Cmd
	for i := range m.form.inputs {
		if i == m.form.focus {
			cmd = m.form.inputs[i].Focus()
		} else {
			m.form.inputs[i].Blur()
		}
	}
	return cmd
}

// saveForm writes the form's profile to the repo-local metadata file.
func (m model) saveForm() (tea.Model, tea.Cmd) {
	previous, _ := m.metadata.Lookup(m.form.author)
	profile := Profile{
		Team:    strings.TrimSpace(m.form.inputs[0].Value()),
		Active:  m.form.active,
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
	return m.switchToAuthors(m.currentPeriod)
}

// formView renders the profile form, centered.
func (m model) formView() string {
	var b strings.Builder
	b.WriteString(previewHeaderStyle.Render(m.form.author) + "\n\n")
	labels := []string{"Team", "Active"}
	for i := range labels {
		cursor := "  "
		if i == m.form.focus {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%-7s ", cursor, labels[i]+":")
		if i < len(m.form.inputs) {
			b.WriteString(m.form.inputs[i].View())
		} else {
			b.WriteString(m.renderActiveRadio())
		}
		b.WriteString("\n")
		if i == 0 && m.form.focus == 0 {
			b.WriteString(m.renderTeamSuggestions())
		}
	}
	if m.form.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.form.errorMsg) + "\n")
	}
	b.WriteString("\n" + footerStyle.Render("enter save · esc cancel · tab next field · space toggle active"))
	box := formStyle.Render(strings.TrimSuffix(b.String(), "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// renderTeamSuggestions lists the known teams plus a row for naming a new
// one, with the highlighted row marked.
func (m model) renderTeamSuggestions() string {
	chosen := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	indent := "          "
	var b strings.Builder
	for i, team := range m.form.knownTeams {
		if i == m.form.suggestionIndex {
			b.WriteString(indent + chosen.Render("> "+team) + "\n")
		} else {
			b.WriteString(indent + "  " + footerStyle.Render(team) + "\n")
		}
	}
	if len(m.form.knownTeams) == m.form.suggestionIndex {
		b.WriteString(indent + chosen.Render("> (new team…)") + "\n")
	} else {
		b.WriteString(indent + "  " + footerStyle.Render("(new team…)") + "\n")
	}
	return b.String()
}

// renderActiveRadio renders the Active field's yes/no radio pair, with the
// selected option filled and highlighted.
func (m model) renderActiveRadio() string {
	yes, no := "( )", "( )"
	if m.form.active {
		yes = "(●)"
	} else {
		no = "(●)"
	}
	chosen := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	other := footerStyle
	if m.form.active {
		return chosen.Render(yes+" yes") + other.Render("   "+no+" no")
	}
	return other.Render(yes+" yes   ") + chosen.Render(no+" no")
}

func (m model) switchToCommits(author string) (tea.Model, tea.Cmd) {
	m.currentAuthor = author

	var items []list.Item
	for _, c := range m.commits {
		if inPeriod(c, m.currentPeriod) && c.Author == author {
			items = append(items, commitItem{commit: c})
		}
	}
	if items == nil {
		items = []list.Item{}
	}
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

// periodLabel titles a list by its period: "March 2024" or "2024".
func periodLabel(period string) string {
	if isYear(period) {
		return period
	}
	return monthLabel(period)
}

// renderPeriodPreview is the contributor list for a period, busiest first.
func renderPeriodPreview(summary PeriodSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s commits by %d authors\n\n", comma(summary.TotalCommits), len(summary.Authors))
	for _, stats := range summary.Authors {
		fmt.Fprintf(&b, "%-24s %6s\n", stats.Author, comma(stats.Commits))
	}
	return b.String()
}

// renderAuthorPreview is the per-author metadata for a month, with the same
// columns as the original script.
func renderAuthorPreview(item authorItem) string {
	var b strings.Builder
	if item.found {
		if item.profile.Team != "" {
			fmt.Fprintf(&b, "%-18s %s\n", "Team", item.profile.Team)
		}
		active := "no"
		if item.profile.Active {
			active = "yes"
		}
		fmt.Fprintf(&b, "%-18s %s\n", "Active", active)
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
		fmt.Fprintf(&b, "%-24s %6s\n", member.Author, comma(member.Commits))
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
	fmt.Fprintf(&b, "%-12s %s\n", "Date", dayLabelWithYear(c.Day))
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
		keys = append(keys, "y years", "enter contributors", "esc/q quit")
	case modeYears:
		keys = append(keys, "m months", "enter contributors", "esc/q quit")
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
	if m.mode == modeForm {
		return m.formView()
	}

	var preview string
	switch m.mode {
	case modeMonths, modeYears:
		if summary, ok := m.selectedPeriod(); ok {
			preview = renderPeriodPreview(summary)
		} else {
			preview = "No commits in this period."
		}
	case modeAuthors:
		if item, ok := m.list.SelectedItem().(authorItem); ok {
			preview = renderAuthorPreview(item)
		} else {
			preview = "No contributors this month."
		}
	case modeTeams:
		if item, ok := m.list.SelectedItem().(teamItem); ok {
			preview = renderTeamPreview(item)
		} else {
			preview = "No teams with commits this month."
		}
	case modeCommits:
		if item, ok := m.list.SelectedItem().(commitItem); ok {
			if m.showDiff {
				preview = diffPaneStyle.Render(m.diffViewport.View())
			} else {
				preview = previewStyle.Render(renderCommitPreview(item))
			}
		} else {
			preview = "No commits this month."
		}
	}

	listBlock := lipgloss.NewStyle().Width(m.width / 3).Render(m.list.View())

	return m.headingView() + "\n" + m.breadcrumbBar() + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, listBlock, m.previewPane(preview)) + "\n" + m.footer()
}

// previewPane pads the preview and clamps it to the terminal height, so a
// month with many contributors cannot push the footer off the top of the
// screen. The diff pane brings its own border and padding, so it is not
// padded again: the extra columns would push its right border off screen.
func (m model) previewPane(preview string) string {
	body := preview
	if !(m.mode == modeCommits && m.showDiff) {
		body = previewStyle.Render(body)
	}
	return lipgloss.NewStyle().MaxHeight(m.height - 3).Render(body)
}

// RunUI runs the TUI over an already-loaded history.
func RunUI(w io.Writer, commits []Commit, months []PeriodSummary, repo *Repo, slug string, metadata Metadata) error {
	if len(months) == 0 {
		fmt.Fprintln(w, "No commits found.")
		return nil
	}
	_, err := tea.NewProgram(newModel(commits, months, repo, slug, metadata), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}
