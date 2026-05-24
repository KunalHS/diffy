# Diffy Plan

Diffy is a terminal-first Git diff tool. The goal is not to replace Git, but to make the common comparison flows easier to choose, inspect, and understand.

No browser UI. No server. Git state mutation is limited to confirmed stash actions.

## Core Principles

- Terminal-native experience.
- Read-only by default.
- Keyboard-first, with mouse interactions where the terminal supports them.
- Show the Git command being run so the tool stays transparent.
- Start with a useful summary, then let the user drill into files.
- Prefer guided choices over forcing the user to remember exact Git syntax.
- Keep command names short, but we will decide them later.

## Interaction Model

The primary interface should be keyboard-first. A user should be able to use the whole tool without touching the mouse.

Expected keyboard actions:

- move up and down through files, commits, and hunks
- open the selected item
- go back
- search/filter within the current view
- open the global command palette
- switch comparison modes
- select one or more files
- quit cleanly

Expected mouse actions:

- click a file to open its diff
- click a commit in file history to open that commit's file diff
- click tabs or mode selectors
- scroll lists and diffs
- select text from the rendered diff using normal terminal behavior

The mouse support should be additive. The tool should still feel fast and complete when used only from the keyboard.

## Global Command Palette

Every TUI screen should support `m` as a global command palette shortcut.

The command palette should open as a popup over the current screen and let the user jump to another mode without restarting Diffy.

Example flows:

```text
start in local changes
press m
choose compare branch
pick target branch
continue in compare view
```

```text
start in compare view
press m
choose file history
pick file
continue in file history view
```

The palette should include all approved workflows:

- local changes
- compare/review branches
- ahead of upstream
- behind upstream
- recent commits
- file compare
- stash
- file history
- commit view

As more workflows are approved, they should be added to the palette.

If the active screen already uses filtering, filtering should either move to a context-specific key or happen inside focused lists. `m` is reserved for the global mode switcher.

## TUI Layout

The target interface should feel like a focused terminal application, not a stream of command output.

Expected layout:

```text
top bar: current mode, compared refs, total + and - counts
left sidebar: files, commits, stashes, or comparison choices depending on mode
main pane: rendered diff
bottom bar: always-visible keyboard shortcuts
```

The bottom shortcut bar is part of the product. It should show the actions that are currently useful in the active screen.

Example shortcut hints:

```text
q quit    m modes     / search      enter open    esc back    j/k move
s sidebar f files     h hunks       w wrap
1 unified  2 split     n next         p previous  ? help
```

The shortcut bar should update by context:

- file list screen: open, filter, select, switch mode, quit
- diff screen: next hunk, previous hunk, wrap, split/unified, back, quit
- file history screen: open commit, filter commits, back, quit
- stash screen: select stash, show files, open diff, back, quit

The sidebar should be resizable if the TUI framework supports it. If resizing is expensive for version 1, use a sensible fixed width and defer resizing.

## Main Workflows

### Local Changes

Show local changes in a clearer way.

Useful Git commands:

```bash
git diff
git diff --staged
git status --short
```

Tool should separate:

- unstaged changes
- staged changes
- untracked files

### Current Branch vs Base

Show what the current branch introduced compared to a base branch.

Useful Git commands:

```bash
git diff main...HEAD
git diff origin/main...HEAD
```

This is the main code-review view.

Branch resolution should be smart:

1. Try the local branch name first.
2. If not found, try `origin/<branch>`.
3. If still not found, show a clear error.

### Current Branch vs Its Remote

Show local branch changes that are not on the upstream remote yet.

Useful Git commands:

```bash
git diff @{u}...HEAD
git log --oneline @{u}..HEAD
```

This helps before pushing.

### Remote vs Local Branch

Show upstream changes that the local branch does not have yet.

Useful Git commands:

```bash
git diff HEAD..@{u}
git log --oneline HEAD..@{u}
```

This helps before pulling or rebasing.

### Two Commits

Compare any two commits.

Useful Git commands:

```bash
git diff commitA commitB
git diff commitA..commitB
```

This helps debug regressions or inspect a fix range.

### Commit Range On Current Branch

Compare the last N commits on the current branch.

Useful Git commands:

```bash
git diff HEAD~3..HEAD
git log --oneline HEAD~3..HEAD
```

This helps answer: what did my last few commits change?

### One File Across Versions

Compare one file across branches, commits, or ranges.

Useful Git commands:

```bash
git diff main...HEAD -- path/to/file
git diff commitA commitB -- path/to/file
```

This helps when the full diff is noisy and only one file matters.

### Working File vs Branch Or Commit

Compare the current working version of a file against a known branch or commit.

Useful Git commands:

```bash
git diff main -- path/to/file
git diff HEAD~1 -- path/to/file
```

This helps inspect a local edit against an older known state.

### Stash Compare

Inspect stashed changes and optionally apply, pop, or drop a stash from the stash screen.

Useful Git commands:

```bash
git stash show -p
git stash apply stash@{n}
git stash pop stash@{n}
git stash drop stash@{n}
```

This helps review saved work before deciding what to do with it. Stash actions are the intentional mutation exception and must require confirmation.

## File History

Show commit history for a specific file, then allow selecting a commit to see only that commit's change for that file.

`diffy history` opens a file selector. `diffy history --file <path>` skips the selector.

Useful Git commands:

```bash
git log --oneline --follow -- path/to/file
git show <commit> -- path/to/file
git diff <commit>^ <commit> -- path/to/file
```

Expected flow:

```text
pick file
show commits touching that file
select commit
show that commit's diff for only that file
open changed files for selected commit
open commit view for all changes in selected commit
```

## Commit View

Show all changes introduced by one commit.

Useful Git command:

```bash
git show <commit>
```

Expected flow:

```text
pick or pass commit
show files changed in the commit
select file to see that file's diff
optionally show the full commit diff
```

## Output Views

The tool should support these output modes:

- TUI: default interactive diff view
- summary: file count, additions, deletions, and per-file additions/deletions
- raw/no-tui: raw/full diff output
- selected file/path: diff restricted to one chosen file/path

For interactive use, the tool can use `fzf` to pick files or commits when available.

## First Version Scope

Version 1 should focus on:

1. Local changes.
2. Current branch vs base branch.
3. Current branch vs upstream remote.
4. Remote vs local branch.
5. Last N commits.
6. Single-file comparisons.
7. Stash comparison.
8. File history for one file.
9. Commit view for one commit.

## Non-Goals

- No browser UI.
- No web server.
- No staging, unstaging, checkout, reset, rebase, or merge.
- No Git mutations except confirmed stash apply, pop, and drop actions inside the stash screen.
- No hidden Git mutations.
- No attempt to replace advanced Git usage.

## Open Decisions

- Final tool name.
- Which external tools are allowed: `fzf`, `delta`, `less`, `bat`, etc.

## Implementation Decisions

- Tech stack: Go.
- First-class interface: interactive terminal TUI.
- Config location: under `.config`.
- Default layout: split view, configurable later.
- Git integration: call the `git` CLI directly; do not use a Git library.
- Build scope: implement the approved command set in one pass rather than limiting to a small MVP.
- Testing: user will test manually for now.
- Install flow: decide later.
- Shortcut keys: choose practical defaults during implementation; they can be changed later.
- Rendering approach: implementation choice, as long as the result is readable and terminal-native.
- Parser details: implementation choice, as long as approved alias behavior is preserved.
- Stash safety: apply, pop, and drop require confirmation and show the exact Git command.
