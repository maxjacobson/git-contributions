Help me convert a fish script to a more robust golang project.

Script is here: /Users/mjacobson/.local/bin/git-contributions

Features:

My requests for you:

- Use golang
- Use charm for TUI
- Add all the features from that script
    - when run from a git repo, it looks at the complete history of the project and shows a list of months during which commits occurred
    - when focusing a month, it shows a preview of the contributions during that month. That is, the list of names of contributors, with commit count, sorted by commit count
    - fuzzy finding of month list is not necessary
    - when drilling down into a month, I can see some more metadata about each contributor's contributions for that month
- Use jj, commit and push early and often
- extend the script so that it still works in non-colocated jj repos. Maybe it uses `jj git root` to find the underlying git repo? Maybe it figures out the corresponding jj commands? up to you
- rewrite this AGENTS.md at your discretion
- Write a README.md
- Add a basic github actions CI and dependabot config
- Add a justfile, with a default command that runs the CLI, and also a test, build, and install command
- write some tests, but not too many, focus on the core logic and what makes sense to test (I'm assuming that testing the UI is annoying and probably not worth it). Lean on me to manually test the UI.
