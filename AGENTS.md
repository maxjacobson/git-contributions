# Notes for coding agents

A Go rewrite of the fish script at ~/.local/bin/git-contributions, which
browses commit history by month in a terminal UI.

## What it does

- Lists every month with commits, newest first
- Focusing a month previews its contributors with commit counts, busiest first
- Drilling into a month shows per-author metadata: commit count, median and
  mean commit size (insertions plus deletions, floored), total insertions and
  deletions
- Selecting an author is stats-only; enter does nothing there
- Dependabot commits are excluded by default; `--include-dependabot` opts in
- In jj workspaces, `jj git root` finds the underlying git dir and all refs
  are walked, since jj's internal git repo has no meaningful HEAD

## Commands

- `just run` — run the CLI (default recipe)
- `just test` — run tests (also: `gofmt -l .`, `go vet ./...`)
- `just build` — build into `bin/`
- `just install` — install into `$GOPATH/bin`

## Structure

- `stats.go` — Commit/AuthorStats/MonthSummary types, aggregation, median/mean
- `parse.go` — parses `git log --pretty=tformat:'%ad;;;%aN;;;' --date=format:'%Y-%m' --shortstat` output
- `repo.go` — jj vs git detection, running git log, dependabot filtering
- `ui.go` — bubbletea TUI (months list → author list, with preview panes)
- `main.go` — flags and wiring

## Conventions

- Commit with jj, atomically, early and often
- Tests cover the core logic, not the UI; MJ tests the UI by hand
- No explanatory code comments
