package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type Git struct {
	Dir string
}

const diffyTUIContextLines = "80"

func (g Git) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if g.Dir != "" {
		cmd.Dir = g.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func (g Git) RunAllowExitOne(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if g.Dir != "" {
		cmd.Dir = g.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return stdout.String(), nil
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return stdout.String(), fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
}

func (g Git) EnsureRepo() error {
	_, err := g.Run("rev-parse", "--show-toplevel")
	return err
}

func (g Git) CurrentBranch() string {
	out, err := g.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "HEAD"
	}
	return strings.TrimSpace(out)
}

func (g Git) Upstream() (string, error) {
	out, err := g.Run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (g Git) ResolveRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("empty ref")
	}
	if strings.HasPrefix(ref, "origin/") || ref == "HEAD" || strings.HasPrefix(ref, "HEAD~") || strings.HasPrefix(ref, "HEAD^") {
		return ref, g.verifyCommit(ref)
	}
	if err := g.verifyFullRef("refs/heads/" + ref); err == nil {
		return ref, nil
	}
	originRef := "origin/" + ref
	if err := g.verifyFullRef("refs/remotes/" + originRef); err == nil {
		return originRef, nil
	}
	if err := g.verifyCommit(ref); err == nil {
		return ref, nil
	}
	return "", fmt.Errorf("could not resolve ref %q locally or as origin/%s", ref, ref)
}

func (g Git) verifyFullRef(ref string) error {
	_, err := g.Run("show-ref", "--verify", "--quiet", ref)
	return err
}

func (g Git) verifyCommit(ref string) error {
	_, err := g.Run("rev-parse", "--verify", ref+"^{commit}")
	return err
}

func (g Git) Branches() ([]string, error) {
	out, err := g.Run("for-each-ref", "--format=%(refname:short)", "refs/heads", "refs/remotes/origin")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "origin/HEAD" || seen[line] {
			continue
		}
		seen[line] = true
		branches = append(branches, line)
	}
	sort.Strings(branches)
	return branches, nil
}

func (g Git) TrackedFiles() ([]string, error) {
	out, err := g.Run("ls-files")
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (g Git) StatusShort() ([]string, error) {
	out, err := g.Run("status", "--short")
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (g Git) UntrackedFiles() ([]string, error) {
	out, err := g.Run("ls-files", "-o", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (g Git) Numstat(args ...string) ([]FileStat, error) {
	out, err := g.Run(append([]string{"diff", "--no-ext-diff", "--numstat"}, args...)...)
	if err != nil {
		return nil, err
	}
	return parseNumstat(out), nil
}

func (g Git) NameOnly(args ...string) ([]string, error) {
	out, err := g.Run(append([]string{"diff", "--no-ext-diff", "--name-only"}, args...)...)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (g Git) Diff(args ...string) (string, error) {
	return g.Run(append([]string{"diff", "--no-ext-diff"}, args...)...)
}

func (g Git) DiffForTUI(args ...string) (string, error) {
	return g.Run(append([]string{"diff", "--no-ext-diff", "--unified=" + diffyTUIContextLines}, args...)...)
}

func (g Git) ShowForTUI(args ...string) (string, error) {
	return g.Run(append([]string{"show", "--no-ext-diff", "--unified=" + diffyTUIContextLines}, args...)...)
}

func (g Git) StashList() ([]StashItem, error) {
	out, err := g.Run("stash", "list")
	if err != nil {
		return nil, err
	}
	var items []StashItem
	for _, line := range nonEmptyLines(out) {
		ref, subject, _ := strings.Cut(line, ": ")
		items = append(items, StashItem{Ref: ref, Subject: subject, Raw: line})
	}
	return items, nil
}

func (g Git) StashNumstat(ref string) ([]FileStat, error) {
	out, err := g.Run("stash", "show", "--no-ext-diff", "--numstat", ref)
	if err != nil {
		return nil, err
	}
	return parseNumstat(out), nil
}

func (g Git) CommitFiles(commit string) ([]FileStat, error) {
	out, err := g.Run("show", "--no-ext-diff", "--numstat", "--format=", commit)
	if err != nil {
		return nil, err
	}
	return parseNumstat(out), nil
}

func (g Git) CommitSubject(commit string) string {
	out, err := g.Run("show", "-s", "--format=%h %s", commit)
	if err != nil {
		return commit
	}
	return strings.TrimSpace(out)
}

func (g Git) CommitHistory(path string) ([]CommitItem, error) {
	out, err := g.Run("log", "--oneline", "--follow", "--", path)
	if err != nil {
		return nil, err
	}
	var items []CommitItem
	for _, line := range nonEmptyLines(out) {
		hash, subject, _ := strings.Cut(line, " ")
		items = append(items, CommitItem{Hash: hash, Subject: subject, Raw: line})
	}
	return items, nil
}

func (g Git) CommitChangedFiles(commit string) ([]string, error) {
	out, err := g.Run("show", "--no-ext-diff", "--name-only", "--format=", commit)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(out), nil
}

func (g Git) UntrackedPatch(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	return g.RunAllowExitOne("diff", "--no-ext-diff", "--no-index", "--", "/dev/null", path)
}

func (g Git) UntrackedPatchForTUI(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	return g.RunAllowExitOne("diff", "--no-ext-diff", "--no-index", "--unified="+diffyTUIContextLines, "--", "/dev/null", path)
}

func (g Git) UntrackedStat(path string) FileStat {
	out, err := g.RunAllowExitOne("diff", "--no-ext-diff", "--no-index", "--numstat", "--", "/dev/null", path)
	if err == nil {
		stats := parseNumstat(out)
		if len(stats) > 0 {
			stat := stats[0]
			stat.Path = path
			stat.Status = "??"
			return stat
		}
	}
	return FileStat{Path: path, Status: "??"}
}

func parseNumstat(out string) []FileStat {
	var stats []FileStat
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		add, binaryAdd := parseCount(parts[0])
		del, binaryDel := parseCount(parts[1])
		stats = append(stats, FileStat{
			Path:   parts[len(parts)-1],
			Add:    add,
			Del:    del,
			Binary: binaryAdd || binaryDel,
		})
	}
	return stats
}

func parseCount(s string) (int, bool) {
	if s == "-" {
		return 0, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, false
}

func nonEmptyLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func mergeStats(groups ...[]FileStat) []FileStat {
	byPath := map[string]*FileStat{}
	var order []string
	for _, stats := range groups {
		for _, stat := range stats {
			if stat.Path == "" {
				continue
			}
			existing, ok := byPath[stat.Path]
			if !ok {
				cp := stat
				byPath[stat.Path] = &cp
				order = append(order, stat.Path)
				continue
			}
			existing.Add += stat.Add
			existing.Del += stat.Del
			if existing.Status == "" {
				existing.Status = stat.Status
			}
			existing.Binary = existing.Binary || stat.Binary
		}
	}
	sort.Strings(order)
	out := make([]FileStat, 0, len(order))
	for _, path := range order {
		out = append(out, *byPath[path])
	}
	return out
}
