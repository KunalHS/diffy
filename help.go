package main

import "fmt"

func Help(cmd Command) string {
	if cmd.Kind == "" {
		return globalHelp()
	}
	switch cmd.Kind {
	case KindLocal:
		return localHelp()
	case KindCompare:
		return compareHelp()
	case KindAhead:
		return simpleHelp("ahead", "Show local commits/diff that are not on the configured upstream.", "diffy ahead")
	case KindBehind:
		return simpleHelp("behind", "Show upstream commits/diff that are not in the local branch.", "diffy behind")
	case KindRecent:
		return recentHelp()
	case KindFile:
		return fileHelp()
	case KindStash:
		return stashHelp()
	case KindHistory:
		return historyHelp()
	case KindCommit:
		return commitHelp()
	default:
		return globalHelp()
	}
}

func globalHelp() string {
	return `Diffy - terminal-first Git diff viewer

Usage:
  diffy
  diffy <command> [args] [options]

Commands:
  diffy                                      Open local uncommitted changes
  diffy local [--unstaged|--staged]          Open local changes explicitly
  diffy <target>                             Compare current branch into target
  diffy <source> <target>                    Compare source branch into target
  diffy compare [source] [target]            Branch review/compare workflow
  diffy review [source] [target]             Alias for compare
  diffy ahead                                Show local commits not on upstream
  diffy behind                               Show upstream commits missing locally
  diffy recent <count>                       Show changes across last N commits
  diffy file                                 Pick a file, then compare with refs
  diffy file <path> <ref>                    Compare working file against a ref
  diffy file <path> <from> <to>              Compare one file across two refs
  diffy stash [stash@{n}]                    Inspect stashes; apply/pop/drop with confirmation
  diffy history [--file path|--path path]    Show commit history for one file
  diffy commit <commit>                      Show all changes in one commit

Shared options:
  --summary              print compact totals and per-file counts, then exit
  --raw, --no-tui        print raw/full diff, then exit
  --unified              start in unified layout
  --split                start in split layout
  --file [path]          restrict to a file, or open a selector in TUI
  --path [path]          alias for --file
  --help                 show help

Examples:
  diffy --summary
  diffy main
  diffy feature/login live --file src/App.ts
  diffy recent 3 --summary
  diffy stash

Notes:
  Known subcommands win over branch shorthand.
  Equivalent aliases cannot be passed together.`
}

func localHelp() string {
	return `diffy local - view uncommitted local work

Usage:
  diffy
  diffy local
  diffy local --unstaged
  diffy local --staged
  diffy local --file [path]
  diffy local --path [path]

Shows unstaged changes, staged changes, and untracked files.`
}

func compareHelp() string {
	return `diffy compare / diffy review - compare branches in review direction

Usage:
  diffy compare
  diffy review
  diffy <target-branch>
  diffy compare <target-branch>
  diffy review <target-branch>
  diffy <source-branch> <target-branch>
  diffy compare <source-branch> <target-branch>
  diffy review <source-branch> <target-branch>

Meaning:
  diffy compare live      current branch -> live
  diffy compare b1 b2     b1 -> b2

Resolution:
  explicit origin/name is used as-is
  otherwise local branch first, then origin/name

Aliases:
  compare and review are equivalent.
  bare branch args are shorthand for compare.`
}

func recentHelp() string {
	return `diffy recent - show changes across the last N commits

Usage:
  diffy recent <count>
  diffy recent <count> --file [path]
  diffy recent <count> --path [path]

Example:
  diffy recent 3`
}

func fileHelp() string {
	return `diffy file - file-first comparison

Usage:
  diffy file
  diffy file <ref>
  diffy file --file <path> <ref>
  diffy file <path> <ref>
  diffy file <path> <from-ref> <to-ref>

Examples:
  diffy file
  diffy file main
  diffy file src/App.ts main
  diffy file src/App.ts HEAD~2 HEAD`
}

func stashHelp() string {
	return `diffy stash - inspect and manage stashes

Usage:
  diffy stash
  diffy stash stash@{2}

Shortcuts in TUI:
  a apply    p pop    d drop/delete    r refresh

Stash apply/pop/drop require confirmation and show the exact Git command.`
}

func historyHelp() string {
	return `diffy history - file history

Usage:
  diffy history
  diffy history --file <path>
  diffy history --path <path>

Without a path, Diffy opens a file selector first.`
}

func commitHelp() string {
	return `diffy commit - inspect one commit

Usage:
  diffy commit <commit>

Shows files changed by the commit and the selected file/full commit diff.`
}

func simpleHelp(name, desc, usage string) string {
	return fmt.Sprintf("diffy %s\n\n%s\n\nUsage:\n  %s\n", name, desc, usage)
}
