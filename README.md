# git-contributions

Browse a repository's commit history by month in a terminal UI.

When run from a git repo, it shows every month during which commits occurred.
Focusing a month previews its contributors with commit counts, sorted by
commit count. Drilling into a month shows per-author metadata: commit count,
median and mean commit size (insertions plus deletions, rounded down), and
total insertions and deletions.

Built with [charm](https://charm.sh) (bubbletea, bubbles, lipgloss).

## Usage

```console
$ git-contributions
```

Inside a jj workspace (colocated or not), the tool uses `jj git root` to find
the underlying git repository and walks all refs, so history committed only in
jj is still visible.

Flags:

- `--include-dependabot` — include commits by dependabot (excluded by default)

## Keys

- `↑`/`↓` — move through the list
- `enter` — drill into the selected month
- `esc` — back out (or quit from the month list)
- `q` or `ctrl+c` — quit
- type to filter the list

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
