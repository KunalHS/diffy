package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func ParseCommand(args []string, cfg Config) (Command, error) {
	opts, positional, err := parseOptions(args, cfg)
	if err != nil {
		return Command{}, err
	}

	if opts.Help && len(positional) == 0 {
		return Command{Options: opts}, nil
	}

	if len(positional) == 0 {
		return Command{Kind: KindLocal, Options: opts}, nil
	}

	head := positional[0]
	if opts.Help {
		return Command{Kind: helpKind(head), Options: opts, Alias: head}, nil
	}

	switch head {
	case "local":
		return parseLocal(positional[1:], opts)
	case "compare", "review":
		return parseCompare(head, positional[1:], opts)
	case "ahead":
		return parseNoArg(KindAhead, positional[1:], opts)
	case "behind":
		return parseNoArg(KindBehind, positional[1:], opts)
	case "recent":
		return parseRecent(positional[1:], opts)
	case "file":
		return parseFile(positional[1:], opts)
	case "stash":
		return parseStash(positional[1:], opts)
	case "history":
		return parseHistory(positional[1:], opts)
	case "commit":
		return parseCommit(positional[1:], opts)
	default:
		return parseBareCompare(positional, opts)
	}
}

func parseOptions(args []string, cfg Config) (Options, []string, error) {
	opts := Options{Layout: cfg.Layout}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help", "-h":
			opts.Help = true
		case "--summary":
			opts.Summary = true
		case "--raw":
			opts.Raw = true
			opts.UsedRaw = true
		case "--no-tui":
			opts.Raw = true
			opts.UsedNoTUI = true
		case "--unified":
			opts.Layout = LayoutUnified
			opts.UsedUnified = true
		case "--split":
			opts.Layout = LayoutSplit
			opts.UsedSplit = true
		case "--unstaged":
			if opts.LocalMode != "" && opts.LocalMode != "unstaged" {
				return opts, nil, errors.New("use only one of --unstaged or --staged")
			}
			opts.LocalMode = "unstaged"
		case "--staged":
			if opts.LocalMode != "" && opts.LocalMode != "staged" {
				return opts, nil, errors.New("use only one of --unstaged or --staged")
			}
			opts.LocalMode = "staged"
		case "--file", "--path":
			if arg == "--file" {
				opts.UsedFile = true
			} else {
				opts.UsedPath = true
			}
			opts.FileFlag = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				if opts.FilePath != "" {
					return opts, nil, errors.New("provide only one file/path restriction")
				}
				opts.FilePath = args[i+1]
				i++
			}
		default:
			if strings.HasPrefix(arg, "--") {
				return opts, nil, fmt.Errorf("unknown option %s", arg)
			}
			positional = append(positional, arg)
		}
	}

	if opts.UsedFile && opts.UsedPath {
		return opts, nil, errors.New("--file and --path are aliases; use only one")
	}
	if opts.UsedRaw && opts.UsedNoTUI {
		return opts, nil, errors.New("--raw and --no-tui are aliases; use only one")
	}
	if opts.UsedUnified && opts.UsedSplit {
		return opts, nil, errors.New("use only one of --unified or --split")
	}
	if opts.Summary && opts.Raw {
		return opts, nil, errors.New("use only one of --summary or --raw/--no-tui")
	}

	return opts, positional, nil
}

func parseLocal(args []string, opts Options) (Command, error) {
	if len(args) > 0 {
		return Command{}, fmt.Errorf("local does not accept positional arguments: %s", strings.Join(args, " "))
	}
	return Command{Kind: KindLocal, Options: opts}, nil
}

func parseCompare(alias string, args []string, opts Options) (Command, error) {
	for _, arg := range args {
		if arg == "compare" || arg == "review" {
			return Command{}, fmt.Errorf("%s and %s are aliases; use only one command alias", alias, arg)
		}
	}
	switch len(args) {
	case 0:
		return Command{Kind: KindCompare, Options: opts, Interactive: true, Alias: alias}, nil
	case 1:
		return Command{Kind: KindCompare, Options: opts, Target: args[0], Alias: alias}, nil
	case 2:
		return Command{Kind: KindCompare, Options: opts, Source: args[0], Target: args[1], Alias: alias}, nil
	default:
		return Command{}, fmt.Errorf("%s accepts at most two branch arguments", alias)
	}
}

func parseBareCompare(args []string, opts Options) (Command, error) {
	for _, arg := range args {
		if knownCommand(arg) {
			return Command{}, fmt.Errorf("%s is a command alias; use it as the first argument or choose a non-ambiguous branch form", arg)
		}
	}
	switch len(args) {
	case 1:
		return Command{Kind: KindCompare, Options: opts, Target: args[0], Alias: "bare"}, nil
	case 2:
		return Command{Kind: KindCompare, Options: opts, Source: args[0], Target: args[1], Alias: "bare"}, nil
	default:
		return Command{}, errors.New("bare diffy accepts zero, one, or two positional arguments")
	}
}

func parseNoArg(kind CommandKind, args []string, opts Options) (Command, error) {
	if len(args) > 0 {
		return Command{}, fmt.Errorf("%s does not accept positional arguments", kind)
	}
	return Command{Kind: kind, Options: opts}, nil
}

func parseRecent(args []string, opts Options) (Command, error) {
	if len(args) != 1 {
		return Command{}, errors.New("recent requires a commit count")
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count < 1 {
		return Command{}, errors.New("recent count must be a positive integer")
	}
	return Command{Kind: KindRecent, Options: opts, Count: count}, nil
}

func parseFile(args []string, opts Options) (Command, error) {
	cmd := Command{Kind: KindFile, Options: opts, Path: opts.FilePath, Interactive: opts.FilePath == ""}
	if opts.FilePath != "" {
		switch len(args) {
		case 0:
			return cmd, nil
		case 1:
			cmd.Ref = args[0]
			return cmd, nil
		case 2:
			cmd.FromRef = args[0]
			cmd.ToRef = args[1]
			return cmd, nil
		default:
			return Command{}, errors.New("file accepts at most two refs when --file/--path provides the file")
		}
	}
	switch len(args) {
	case 0:
		return cmd, nil
	case 1:
		cmd.Ref = args[0]
		return cmd, nil
	case 2:
		cmd.Path = args[0]
		cmd.Ref = args[1]
		return cmd, nil
	case 3:
		cmd.Path = args[0]
		cmd.FromRef = args[1]
		cmd.ToRef = args[2]
		return cmd, nil
	default:
		return Command{}, errors.New("file requires: diffy file [path] <ref> OR diffy file [path] <from-ref> <to-ref>")
	}
}

func parseStash(args []string, opts Options) (Command, error) {
	if len(args) > 1 {
		return Command{}, errors.New("stash accepts at most one stash ref")
	}
	cmd := Command{Kind: KindStash, Options: opts}
	if len(args) == 1 {
		cmd.StashRef = args[0]
	}
	return cmd, nil
}

func parseHistory(args []string, opts Options) (Command, error) {
	if len(args) > 0 {
		return Command{}, errors.New("history uses --file/--path for a file path")
	}
	return Command{Kind: KindHistory, Options: opts, Path: opts.FilePath, Interactive: opts.FilePath == ""}, nil
}

func parseCommit(args []string, opts Options) (Command, error) {
	if len(args) != 1 {
		return Command{}, errors.New("commit requires one commit ref")
	}
	return Command{Kind: KindCommit, Options: opts, Commit: args[0]}, nil
}

func helpKind(s string) CommandKind {
	switch s {
	case "local":
		return KindLocal
	case "compare", "review":
		return KindCompare
	case "ahead":
		return KindAhead
	case "behind":
		return KindBehind
	case "recent":
		return KindRecent
	case "file":
		return KindFile
	case "stash":
		return KindStash
	case "history":
		return KindHistory
	case "commit":
		return KindCommit
	default:
		return ""
	}
}
