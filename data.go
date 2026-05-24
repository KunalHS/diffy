package main

import (
	"fmt"
	"strings"
)

type Controller struct {
	Git Git
}

func (c Controller) Build(cmd Command) (ViewData, error) {
	switch cmd.Kind {
	case KindLocal:
		return c.buildLocal(cmd)
	case KindCompare:
		return c.buildCompare(cmd)
	case KindAhead:
		return c.buildAhead(cmd)
	case KindBehind:
		return c.buildBehind(cmd)
	case KindRecent:
		return c.buildRecent(cmd)
	case KindFile:
		return c.buildFile(cmd)
	case KindStash:
		return c.buildStash(cmd)
	case KindHistory:
		return c.buildHistory(cmd)
	case KindCommit:
		return c.buildCommit(cmd)
	default:
		return ViewData{}, fmt.Errorf("unsupported command %s", cmd.Kind)
	}
}

func (c Controller) Raw(cmd Command) (string, error) {
	view, err := c.Build(cmd)
	if err != nil {
		return "", err
	}
	return view.Diff, nil
}

func (c Controller) Summary(cmd Command) (string, error) {
	view, err := c.Build(cmd)
	if err != nil {
		return "", err
	}
	return FormatSummary(view), nil
}

func (c Controller) buildLocal(cmd Command) (ViewData, error) {
	if err := c.Git.EnsureRepo(); err != nil {
		return ViewData{}, err
	}

	mode := cmd.Options.LocalMode
	if mode == "" {
		mode = "unstaged"
	}

	var stats []FileStat
	var diffParts []string
	var gitCommand string
	path := cmd.Options.FilePath

	if mode == "staged" {
		args := []string{"--staged"}
		if path != "" {
			args = append(args, "--", path)
		}
		staged, err := c.Git.Numstat(args...)
		if err != nil {
			return ViewData{}, err
		}
		stats = staged
		diff, err := c.Git.Diff(args...)
		if err != nil {
			return ViewData{}, err
		}
		diffParts = append(diffParts, diff)
		gitCommand = "git diff --no-ext-diff --staged"
	} else {
		args := []string{}
		if path != "" {
			args = append(args, "--", path)
		}
		unstaged, err := c.Git.Numstat(args...)
		if err != nil {
			return ViewData{}, err
		}
		diff, err := c.Git.Diff(args...)
		if err != nil {
			return ViewData{}, err
		}
		stats = unstaged
		diffParts = append(diffParts, diff)
		gitCommand = "git diff --no-ext-diff"

		untracked, err := c.Git.UntrackedFiles()
		if err == nil {
			var untrackedStats []FileStat
			for _, file := range untracked {
				if path != "" && file != path {
					continue
				}
				untrackedStats = append(untrackedStats, c.Git.UntrackedStat(file))
				patch, patchErr := c.Git.UntrackedPatch(file)
				if patchErr == nil && strings.TrimSpace(patch) != "" {
					diffParts = append(diffParts, patch)
				}
			}
			stats = mergeStats(stats, untrackedStats)
		}
	}

	if cmd.Options.FileFlag && path == "" {
		stats = markSelectable(stats)
	}

	title := "Local changes"
	if mode == "staged" {
		title = "Local staged changes"
	}
	return ViewData{
		Title:          title,
		Subtitle:       "uncommitted workspace changes",
		GitCommand:     gitCommand,
		Files:          stats,
		Diff:           strings.TrimSpace(strings.Join(diffParts, "\n")),
		Mode:           KindLocal,
		RestrictedPath: path,
	}, nil
}

func (c Controller) buildCompare(cmd Command) (ViewData, error) {
	if err := c.Git.EnsureRepo(); err != nil {
		return ViewData{}, err
	}
	if cmd.Target == "" {
		return ViewData{}, errNeedsPicker("branch")
	}
	target, err := c.Git.ResolveRef(cmd.Target)
	if err != nil {
		return ViewData{}, err
	}
	source := "HEAD"
	if cmd.Source != "" {
		source, err = c.Git.ResolveRef(cmd.Source)
		if err != nil {
			return ViewData{}, err
		}
	}
	path := cmd.Options.FilePath
	args := []string{target + "..." + source}
	if path != "" {
		args = append(args, "--", path)
	}
	stats, err := c.Git.Numstat(args...)
	if err != nil {
		return ViewData{}, err
	}
	diff, err := c.Git.Diff(args...)
	if err != nil {
		return ViewData{}, err
	}
	commits, _ := c.logRange(target, source)
	title := "Compare"
	if cmd.Alias == "review" {
		title = "Review"
	}
	return ViewData{
		Title:           title,
		Subtitle:        fmt.Sprintf("%s -> %s", displaySource(cmd, source), target),
		GitCommand:      "git diff --no-ext-diff " + strings.Join(args, " "),
		Files:           stats,
		Commits:         commits,
		Diff:            diff,
		Mode:            KindCompare,
		RestrictedPath:  path,
		SourceIsCurrent: source == "HEAD" || source == c.Git.CurrentBranch(),
	}, nil
}

func (c Controller) buildAhead(cmd Command) (ViewData, error) {
	upstream, err := c.Git.Upstream()
	if err != nil {
		return ViewData{}, fmt.Errorf("current branch has no upstream")
	}
	args := []string{upstream + "...HEAD"}
	stats, err := c.Git.Numstat(args...)
	if err != nil {
		return ViewData{}, err
	}
	diff, err := c.Git.Diff(args...)
	if err != nil {
		return ViewData{}, err
	}
	commits, _ := c.logRange(upstream, "HEAD")
	return ViewData{
		Title:           "Ahead of upstream",
		Subtitle:        c.Git.CurrentBranch() + " -> " + upstream,
		GitCommand:      "git diff --no-ext-diff " + upstream + "...HEAD",
		Files:           stats,
		Commits:         commits,
		Diff:            diff,
		Mode:            KindAhead,
		SourceIsCurrent: true,
	}, nil
}

func (c Controller) buildBehind(cmd Command) (ViewData, error) {
	upstream, err := c.Git.Upstream()
	if err != nil {
		return ViewData{}, fmt.Errorf("current branch has no upstream")
	}
	args := []string{"HEAD.." + upstream}
	stats, err := c.Git.Numstat(args...)
	if err != nil {
		return ViewData{}, err
	}
	diff, err := c.Git.Diff(args...)
	if err != nil {
		return ViewData{}, err
	}
	commits, _ := c.logRange("HEAD", upstream)
	return ViewData{
		Title:      "Behind upstream",
		Subtitle:   upstream + " -> " + c.Git.CurrentBranch(),
		GitCommand: "git diff --no-ext-diff HEAD.." + upstream,
		Files:      stats,
		Commits:    commits,
		Diff:       diff,
		Mode:       KindBehind,
	}, nil
}

func (c Controller) buildRecent(cmd Command) (ViewData, error) {
	base := fmt.Sprintf("HEAD~%d", cmd.Count)
	path := cmd.Options.FilePath
	args := []string{base + "..HEAD"}
	if path != "" {
		args = append(args, "--", path)
	}
	stats, err := c.Git.Numstat(args...)
	if err != nil {
		return ViewData{}, err
	}
	diff, err := c.Git.Diff(args...)
	if err != nil {
		return ViewData{}, err
	}
	commits, _ := c.logRange(base, "HEAD")
	return ViewData{
		Title:          fmt.Sprintf("Recent %d commits", cmd.Count),
		Subtitle:       base + "..HEAD",
		GitCommand:     "git diff --no-ext-diff " + strings.Join(args, " "),
		Files:          stats,
		Commits:        commits,
		Diff:           diff,
		Mode:           KindRecent,
		RestrictedPath: path,
	}, nil
}

func (c Controller) buildFile(cmd Command) (ViewData, error) {
	var args []string
	var title string
	if cmd.Ref != "" {
		ref, err := c.Git.ResolveRef(cmd.Ref)
		if err != nil {
			return ViewData{}, err
		}
		args = []string{ref, "--", cmd.Path}
		title = "File vs " + ref
	} else {
		from, err := c.Git.ResolveRef(cmd.FromRef)
		if err != nil {
			return ViewData{}, err
		}
		to, err := c.Git.ResolveRef(cmd.ToRef)
		if err != nil {
			return ViewData{}, err
		}
		args = []string{from, to, "--", cmd.Path}
		title = "File " + from + " -> " + to
	}
	stats, err := c.Git.Numstat(args...)
	if err != nil {
		return ViewData{}, err
	}
	diff, err := c.Git.Diff(args...)
	if err != nil {
		return ViewData{}, err
	}
	return ViewData{
		Title:          title,
		Subtitle:       cmd.Path,
		GitCommand:     "git diff --no-ext-diff " + strings.Join(args, " "),
		Files:          stats,
		Diff:           diff,
		Mode:           KindFile,
		RestrictedPath: cmd.Path,
	}, nil
}

func (c Controller) buildStash(cmd Command) (ViewData, error) {
	stashes, err := c.Git.StashList()
	if err != nil {
		return ViewData{}, err
	}
	ref := cmd.StashRef
	if ref == "" && len(stashes) > 0 {
		ref = stashes[0].Ref
	}
	if ref == "" {
		return ViewData{Title: "Stash", Subtitle: "no stashes", Mode: KindStash, Message: "No stashes found."}, nil
	}
	stats, _ := c.Git.StashNumstat(ref)
	diff, err := c.Git.Run("stash", "show", "--no-ext-diff", "-p", ref)
	if err != nil {
		return ViewData{}, err
	}
	return ViewData{
		Title:      "Stash",
		Subtitle:   ref,
		GitCommand: "git stash show --no-ext-diff -p " + ref,
		Files:      stats,
		Stashes:    stashes,
		Diff:       diff,
		Mode:       KindStash,
	}, nil
}

func (c Controller) buildHistory(cmd Command) (ViewData, error) {
	if cmd.Path == "" {
		return ViewData{}, errNeedsPicker("file")
	}
	commits, err := c.Git.CommitHistory(cmd.Path)
	if err != nil {
		return ViewData{}, err
	}
	diff := ""
	title := "History"
	subtitle := cmd.Path
	gitCommand := "git log --oneline --follow -- " + cmd.Path
	if len(commits) > 0 {
		out, err := c.Git.Run("show", "--no-ext-diff", commits[0].Hash, "--", cmd.Path)
		if err == nil {
			diff = out
			gitCommand = "git show --no-ext-diff " + commits[0].Hash + " -- " + cmd.Path
		}
	}
	return ViewData{
		Title:          title,
		Subtitle:       subtitle,
		GitCommand:     gitCommand,
		Commits:        commits,
		Diff:           diff,
		Mode:           KindHistory,
		RestrictedPath: cmd.Path,
	}, nil
}

func (c Controller) buildCommit(cmd Command) (ViewData, error) {
	files, err := c.Git.CommitFiles(cmd.Commit)
	if err != nil {
		return ViewData{}, err
	}
	diff, err := c.Git.Run("show", "--no-ext-diff", cmd.Commit)
	if err != nil {
		return ViewData{}, err
	}
	return ViewData{
		Title:      "Commit",
		Subtitle:   c.Git.CommitSubject(cmd.Commit),
		GitCommand: "git show --no-ext-diff " + cmd.Commit,
		Files:      files,
		Diff:       diff,
		Mode:       KindCommit,
	}, nil
}

func (c Controller) logRange(from, to string) ([]CommitItem, error) {
	out, err := c.Git.Run("log", "--oneline", from+".."+to)
	if err != nil {
		return nil, err
	}
	var commits []CommitItem
	for _, line := range nonEmptyLines(out) {
		hash, subject, _ := strings.Cut(line, " ")
		commits = append(commits, CommitItem{Hash: hash, Subject: subject, Raw: line})
	}
	return commits, nil
}

func displaySource(cmd Command, resolved string) string {
	if cmd.Source == "" {
		return "current branch"
	}
	return resolved
}

func markSelectable(stats []FileStat) []FileStat {
	for i := range stats {
		stats[i].Selected = true
	}
	return stats
}

type pickerNeededError struct {
	Kind string
}

func (e pickerNeededError) Error() string {
	return e.Kind + " selector required"
}

func errNeedsPicker(kind string) error {
	return pickerNeededError{Kind: kind}
}
