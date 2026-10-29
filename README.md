# git-contributions

Browse a repository's commit history by month in a terminal UI.

When run from a git repo, it shows every month during which commits occurred.
Focusing a month previews its contributors with commit counts, sorted by
commit count. Drilling into a month, then into a contributor, lists that
contributor's individual commits for the month, newest first, and entering a
commit shows its date, hash, and size. The author view shows per-author
metadata, split into merge commits and non-merge commits, followed by the
non-merge size statistics: median and mean commit size, plus total insertions
and deletions. Merge commits have no diff of their own, so they are excluded
from the size statistics.

Built with [charm](https://charm.sh) (bubbletea, bubbles, lipgloss).

**This project is vibecoded**: it was written by an AI coding agent
([Crush](https://charm.land/blog/crush)) with MJ as the client and tech
direction. See `AGENTS.md` for the working notes.

## Usage

```console
$ git-contributions
```

Git finds `git-contributions` on your `PATH`, so you can also run it as
`git contributions` from inside any repo.

Inside a jj workspace (colocated or not), the tool uses `jj git root` to find
the underlying git repository and walks the ancestry of the working copy
commit, matching what `jj log` shows. Empty commits (the working copy commits
jj scatters along the way) are not counted, and names are shown as specified
in `.mailmap`.

Flags:

- `--include-dependabot` — include commits by dependabot (excluded by default)

## Keys

- `↑`/`↓` — move through the list
- `enter` — drill into the selected month, contributor, or commit
- `o` — in the commit view, open the commit on GitHub (GitHub repos only)
- `s` — in the commit view, show the diff with `git show` or `jj show`,
  using your own pager setup
- `esc` — back out (or quit from the month list)
- `q` or `ctrl+c` — quit

## Development

A [justfile](justfile) wraps the common tasks:

- `just run` — run the CLI (the default recipe)
- `just test` — run the test suite
- `just fmt` — format with gofmt
- `just lint` — run go vet and check formatting
- `just build` — build into `bin/`
- `just install` — install into `$GOPATH/bin`

Tests cover the core logic (log parsing, aggregation, median/mean math); the
UI is best tested by hand.
