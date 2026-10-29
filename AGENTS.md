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
- In jj workspaces, `jj git root` finds the underlying git dir and the tool
  walks the ancestry of the working copy commit: `--all` would count the
  abandoned and rewritten versions jj keeps alive via refs/jj/keep, which
  inflates counts badly. Empty commits are skipped (merge commits are kept,
  they just have no stat line)
- Names are mailmapped: git runs from the workspace root (and gets
  `mailmap.file` passed explicitly when .mailmap exists there), since git
  does not discover .mailmap when --git-dir points inside .jj
- Merge commits count toward commit totals but are excluded from the size
  statistics: they have no stat line, and counting their zero size drove
  medians to 0 in merge-heavy repos like rails/rails

## Commands

- `just run` — run the CLI (default recipe)
- `just test` — run tests
- `just fmt` — format with gofmt
- `just lint` — go vet plus a gofmt check
- `just build` — build into `bin/`
- `just install` — install into `$GOPATH/bin`

## Structure

- `stats.go` — Commit/AuthorStats/MonthSummary types, aggregation, median/mean
- `parse.go` — parses `git log --pretty=tformat:'%ad;;;%aN;;;%P' --date=format:'%Y-%m' --shortstat` output
- `repo.go` — jj vs git detection, running git log, dependabot filtering
- `ui.go` — bubbletea TUI (months list → author list, with preview panes)
- `main.go` — flags and wiring

## Conventions

- Commit with jj, atomically, early and often
- To publish: describe the commit, then `jj bookmark set main -r @` and
  `jj git push` (after `jj new`, `main -r @-` instead). Setting the bookmark
  to `@-` before describing has twice left the newest commit off main
- Tests cover the core logic, not the UI; MJ tests the UI by hand
- No explanatory code comments
