# git-contributions

Browse a repository's commit history by month in a terminal UI, built with
[charm](https://charm.sh) (bubbletea, bubbles, lipgloss).

**This project is vibecoded**: it was written by an AI coding agent
([Crush](https://charm.land/blog/crush)) with MJ as the client and tech
direction. See `AGENTS.md` for the working notes.

## What it does

Three levels, each with a preview pane on the right:

1. **Months**, newest first, with the contributors of the focused month and
   their commit counts, busiest first
2. **Contributors** of the selected month, with commit metadata: merge and
   non-merge commit counts, then median and mean commit size (insertions
   plus deletions, floored), total insertions and deletions
3. **Commits** of the selected contributor, newest first; `enter` swaps the
   metadata preview for the commit's own colorized rendering in a bordered,
   scrollable pane

Merge commits count toward commit totals but are excluded from the size
statistics — they have no diff of their own, and counting their zero size
drove medians to 0 in merge-heavy repos like rails/rails. Empty commits are
not counted at all.

## Contributor metadata

You can annotate contributors with teams and active status.
The data lives entirely outside the repo's history, in a repo-local file:

- jj repos: `.jj/git-contributions-metadata`
- git repos: `.git/git-contributions-metadata`

Edit it by hand, or press `e` on a contributor in the contributor view to
fill in the form (tab moves between fields, ↓/↑ picks a known team or "(new
team…)" to type one, space or ←/→ flips the yes/no radio, enter saves, esc
cancels):
```json
{
  "Akira Matsuda": {
    "active": true,
    "team": "Core",
    "aliases": ["akira"]
  }
}
```

All fields are optional, and active defaults to true for contributors with
no profile yet. Profiles are keyed by the mailmapped display name, and
`aliases` lets other names resolve to the same profile. In the
contributor view, `t` toggles between the people list and a teams list with
pooled stats and a roster; `enter` on a team opens that team's commits for
the month, with each commit's author named. Contributors without a team land
in a `(no team)` bucket. A missing or malformed file just means no metadata.

## Usage

```console
$ git-contributions
```

Git finds `git-contributions` on your `PATH`, so you can also run it as
`git contributions` from inside any repo.

Flags:

- `--include-dependabot` — include commits by dependabot (excluded by default)

## How history is read

- Plain git repos: `git log` from the current checkout, as usual
- jj workspaces (colocated or not): `jj git root` finds the underlying git
  repository, and the ancestry of the working copy commit is walked,
  matching what `jj log` shows. Walking all refs would be wrong: jj keeps
  `refs/jj/keep` refs for rewritten and abandoned commit versions, which
  inflates counts badly
- Empty commits (the working copy commits jj scatters along the way) are
  skipped; names are shown as specified in `.mailmap` (git is invoked from
  the workspace root and given `mailmap.file` explicitly, since it does not
  discover `.mailmap` when `--git-dir` points inside `.jj`)

## Keys

- `↑`/`↓` and `j`/`k` — navigate the list; scroll the commit diff once it's
  shown with `enter`
- `enter` — drill into the selected month or contributor; in the commit
  view, show or hide the commit's colorized diff
- `e` — in the contributor view, edit the selected contributor's profile
  (team, active) in a form; saved to the repo-local metadata file
- `t` — in the contributor view, toggle between people and teams
- `space`/`pgup`/`pgdn` — page the commit diff
- `o` — in the commit view, open the commit on GitHub (GitHub repos only)
- `esc` — close the commit diff if shown, otherwise back out (or quit from
  the month list)
- `q` — same as `esc`; quits from the month list
- `ctrl+c` — quit from anywhere

Backing out restores the selection you left behind; drilling in starts at
the top.

## The diff pane

The commit diff is rendered by your own tools, not reimplemented: `jj show`
in jj repos, `git show` in git repos, piped through `delta` when one is
configured (the invocation is read from `core.pager`, so your theme and
options carry over). Delta is given the pane width so line backgrounds paint
edge to edge; its erase-to-end-of-line padding is expanded into literal
spaces because bubbletea does not honor it. The diff is captured once per
commit, on demand, and refetched when the selection or terminal size
changes.

## Development

A [justfile](justfile) wraps the common tasks:

- `just run` — run the CLI (the default recipe)
- `just test` — run the test suite
- `just fmt` — format with gofmt
- `just lint` — run go vet and check formatting
- `just build` — build into `bin/`
- `just install` — install into `$GOPATH/bin`

Tests cover the core logic (log parsing, aggregation, median/mean math);
the UI is best tested by hand.
