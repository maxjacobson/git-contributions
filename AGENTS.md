# Notes for coding agents

A Go rewrite of the fish script at ~/.local/bin/git-contributions: a terminal
UI for browsing commit history by month, built with charm (bubbletea,
bubbles, lipgloss).

## What it does

Three drill-down levels, each with a preview pane on the right:

1. Months, newest first; the preview lists the month's contributors with
   commit counts, busiest first
2. Contributors of the selected month; the preview shows merge and non-merge
   commit counts, then median and mean commit size (insertions plus
   deletions, floored), total insertions and deletions
3. Commits of the selected contributor, newest first; the preview shows
   commit metadata, and `enter` swaps it for the commit's own colorized
   rendering in a bordered, scrollable pane; `o` opens the commit on GitHub
   when origin points at github.com (macOS `open`, Linux `xdg-open`)

Behavioral decisions worth remembering:

- Merge commits count toward totals but are excluded from size statistics:
  they have no stat line, and their zero size drove medians to 0 in
  merge-heavy repos like rails/rails. The author preview splits merge and
  non-merge counts so both facts are visible
- Empty commits are skipped entirely (jj scatters empty working copy commits)
- In jj workspaces, `jj git root` finds the underlying git dir and the tool
  walks the ancestry of the working copy commit. Never use `--all` there:
  jj keeps `refs/jj/keep` refs for rewritten/abandoned versions and counts
  inflate badly
- Names are mailmapped: git runs from the workspace root and gets
  `mailmap.file` passed explicitly, since git does not discover `.mailmap`
  when `--git-dir` points inside `.jj`
- Dependabot commits are excluded by default; `--include-dependabot` opts in
- Cursor position is restored when backing out; drilling in starts at the
  top (bubbles list preserves its index across SetItems, so every
  transition selects explicitly)
- The list's own filtering (`/`) is disabled; MJ does not want it
- Contributor metadata lives repo-locally, outside history:
  `.jj/git-contributions-metadata` (jj) or `.git/git-contributions-metadata`
  (git). JSON, keyed by mailmapped name, with active/team/aliases; the
  README has the schema. `e` in the contributor view opens a form to edit a
  profile (tab and shift+tab cycle fields; on team, ↓/↑ picks from known
  teams or "(new team…)"; space, ←/→, or h/l flips the yes/no radio; enter
  saves; esc cancels).
  `t` toggles the teams view; unassigned contributors
  land in a `(no team)` bucket. `enter` on a team opens that team's commits
  for the month, each labeled with its author; esc backs out to the teams
  view, then the contributor view

## The diff pane

Do not reimplement diff rendering. Capture the output of the user's own
tools and display it:

- `jj show -r <hash> --no-pager --color never` in jj repos, `git --no-pager
  show <hash>` in git repos, piped through delta when one is configured
- The delta invocation is read from `core.pager` when it starts with delta,
  with bare `delta` on PATH as the fallback, so MJ's theme and options carry
  over; delta is given `--width` so line backgrounds paint edge to
  edge
- Delta pads with `ESC[0K`, which terminals render via the current
  background but bubbletea does not honor; expand those sequences into
  literal spaces at the pane width before display
- The diff is captured lazy per commit (on `enter`), refetched when the
  selection or terminal size changes
- `enter` toggles the pane, `esc` closes it before leaving the view, and
  `↑`/`↓`, `j`/`k` always do the same thing: navigate lists, scroll the
  open diff

## Commands

- `just run` — run the CLI (default recipe)
- `just test` — run tests
- `just fmt` — format with gofmt
- `just lint` — go vet plus a gofmt check
- `just build` — build into `bin/`
- `just install` — install into `$GOPATH/bin`

## Structure

- `stats.go` — Commit/AuthorStats/MonthSummary types, aggregation, median/mean
- `metadata.go` — contributor profiles: load, aliases, lookup
- `parse.go` — parses the `git log --shortstat` output format
- `repo.go` — jj vs git detection, running git/jj, GitHub slug from origin
- `ui.go` — the TUI: month/contributor/team/commit views, preview panes,
  actions bar, diff viewport, profile form
- `main.go` — flags and wiring

## Conventions

- Commit with jj, atomically, early and often
- To publish: describe the commit, then `jj bookmark set main -r @` and
  `jj git push` (after `jj new`, `main -r @-` instead). Setting the bookmark
  to `@-` before describing has twice left the newest commit off main
- Tests cover the core logic, not the UI; MJ tests the UI by hand and gives
  fast feedback — for UI iterations, build and let him try before committing
- Ask before writing code; MJ weighs in on UX decisions like keybindings
- Use the user's own tools rather than reimplementing their behavior
- No explanatory code comments
