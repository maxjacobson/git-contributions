package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type countedMsg struct {
	total int
	err   error
}

type walkMsg struct {
	seen    int
	oldest  string
	commits []Commit
	err     error
	final   bool
}

type loaderModel struct {
	repo    *Repo
	spinner spinner.Model
	bar     progress.Model
	total   int
	seen    int
	oldest  string
	counted bool
	aborted bool
	err     error
	commits []Commit
	walker  *walker
	width   int
	height  int
}

func newLoaderModel(repo *Repo) loaderModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return loaderModel{
		repo:    repo,
		spinner: s,
		bar:     progress.New(progress.WithDefaultGradient()),
	}
}

func (m loaderModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		total, err := m.repo.CountCommits()
		return countedMsg{total: total, err: err}
	})
}

func (m loaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width >= 20 {
			m.bar.Width = min(msg.Width-8, 60)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			if m.walker != nil {
				m.walker.kill()
			}
			m.aborted = true
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case countedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.counted = true
		m.total = msg.total
		if msg.total == 0 {
			return m, tea.Quit
		}
		m.walker = newWalker(m.repo)
		return m, tea.Batch(m.walker.start(), listenWalk(m.walker.ch))

	case walkMsg:
		m.seen = msg.seen
		m.oldest = msg.oldest
		if msg.final {
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.commits = msg.commits
			}
			return m, tea.Quit
		}
		return m, listenWalk(m.walker.ch)
	}
	return m, nil
}

func (m loaderModel) View() string {
	if m.err != nil {
		return m.place(errorStyle.Render("error: " + m.err.Error()))
	}
	if !m.counted {
		return m.place(m.spinner.View() + " counting commits…")
	}
	frac := 1.0
	if m.total > 0 {
		frac = min(float64(m.seen)/float64(m.total), 1)
	}
	label := fmt.Sprintf("%s of %s commits", comma(m.seen), comma(m.total))
	if len(m.oldest) >= 7 {
		label += " · back to " + m.oldest[:7]
	}
	content := m.bar.ViewAs(frac) + "\n\n" + footerStyle.Render(label)
	return m.place(content)
}

func (m loaderModel) place(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// loadCommits runs the loading view over the repo's history walk and returns
// the parsed commits, whether the user aborted, and any error.
func loadCommits(repo *Repo) (commits []Commit, aborted bool, err error) {
	p := tea.NewProgram(newLoaderModel(repo), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	m, ok := final.(loaderModel)
	if !ok {
		return nil, false, fmt.Errorf("unexpected loading model")
	}
	return m.commits, m.aborted, m.err
}

// walker streams git log's output as it is produced, sending progress
// messages on ch and a final one with the parsed commits.
type walker struct {
	ch  chan walkMsg
	cmd *exec.Cmd
}

func newWalker(repo *Repo) *walker {
	cmd := exec.Command("git", repo.logArgs()...)
	cmd.Dir = repo.Dir
	return &walker{ch: make(chan walkMsg, 8), cmd: cmd}
}

func (w *walker) start() tea.Cmd {
	return func() tea.Msg {
		var stderr bytes.Buffer
		w.cmd.Stderr = &stderr
		stdout, err := w.cmd.StdoutPipe()
		if err != nil {
			w.ch <- walkMsg{err: fmt.Errorf("git log failed: %w", err), final: true}
			return nil
		}
		if err := w.cmd.Start(); err != nil {
			w.ch <- walkMsg{err: fmt.Errorf("git log failed: %w", err), final: true}
			return nil
		}
		go w.read(stdout, &stderr)
		return nil
	}
}

func (w *walker) kill() {
	if w.cmd.Process != nil {
		w.cmd.Process.Kill()
	}
}

func (w *walker) read(stdout io.Reader, stderr *bytes.Buffer) {
	p := &logParser{}
	last := time.Now()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		p.Feed(scanner.Text())
		if p.err != nil {
			break
		}
		if p.Seen%500 == 0 || time.Since(last) > 100*time.Millisecond {
			last = time.Now()
			w.ch <- walkMsg{seen: p.Seen, oldest: p.Oldest}
		}
	}
	if p.err != nil {
		w.kill()
		w.cmd.Wait()
		w.ch <- walkMsg{seen: p.Seen, oldest: p.Oldest, err: p.err, final: true}
		return
	}
	if err := scanner.Err(); err != nil {
		w.ch <- walkMsg{err: fmt.Errorf("git log failed: %w", err), final: true}
		return
	}
	if err := w.cmd.Wait(); err != nil {
		w.ch <- walkMsg{err: fmt.Errorf("git log failed: %s", strings.TrimSpace(stderr.String())), final: true}
		return
	}
	commits, err := p.Finish()
	w.ch <- walkMsg{seen: p.Seen, oldest: p.Oldest, commits: commits, err: err, final: true}
}

func listenWalk(ch chan walkMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}
