package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestControllerLocalCompareRecentHistoryCommitAndStash(t *testing.T) {
	repo := newFixtureRepo(t)
	g := Git{Dir: repo}
	c := Controller{Git: g}

	appendFile(t, repo, "app.txt", "local change\n")
	writeFile(t, repo, "untracked.txt", "new file\n")

	local, err := c.Build(Command{Kind: KindLocal, Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("local build failed: %v", err)
	}
	if !hasPath(local.Files, "app.txt") || !hasPath(local.Files, "untracked.txt") {
		t.Fatalf("local files = %#v, want app.txt and untracked.txt", local.Files)
	}
	if !strings.Contains(local.Diff, "local change") || !strings.Contains(local.Diff, "untracked.txt") {
		t.Fatalf("local diff missing expected content:\n%s", local.Diff)
	}

	runGit(t, repo, "add", "app.txt")
	staged, err := c.Build(Command{Kind: KindLocal, Options: Options{Layout: LayoutSplit, LocalMode: "staged"}})
	if err != nil {
		t.Fatalf("staged build failed: %v", err)
	}
	if !strings.Contains(staged.Diff, "local change") {
		t.Fatalf("staged diff missing change:\n%s", staged.Diff)
	}
	runGit(t, repo, "commit", "-m", "local commit")

	compare, err := c.Build(Command{Kind: KindCompare, Target: "main", Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("compare build failed: %v", err)
	}
	if compare.GitCommand != "git diff --no-ext-diff --unified=80 main...HEAD" {
		t.Fatalf("compare git command = %q", compare.GitCommand)
	}
	if !hasPath(compare.Files, "app.txt") {
		t.Fatalf("compare files = %#v, want app.txt", compare.Files)
	}

	rawCompare, err := c.Build(Command{Kind: KindCompare, Target: "main", Options: Options{Layout: LayoutSplit, Raw: true}})
	if err != nil {
		t.Fatalf("raw compare build failed: %v", err)
	}
	if rawCompare.GitCommand != "git diff --no-ext-diff main...HEAD" {
		t.Fatalf("raw compare git command = %q", rawCompare.GitCommand)
	}

	recent, err := c.Build(Command{Kind: KindRecent, Count: 1, Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("recent build failed: %v", err)
	}
	if !strings.Contains(recent.Diff, "local change") {
		t.Fatalf("recent diff missing expected content:\n%s", recent.Diff)
	}

	fileView, err := c.Build(Command{Kind: KindFile, Path: "app.txt", Ref: "main", Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("file build failed: %v", err)
	}
	if !strings.Contains(fileView.Diff, "local change") {
		t.Fatalf("file diff missing expected content:\n%s", fileView.Diff)
	}

	history, err := c.Build(Command{Kind: KindHistory, Path: "app.txt", Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("history build failed: %v", err)
	}
	if len(history.Commits) < 2 {
		t.Fatalf("history commits = %#v, want at least two", history.Commits)
	}
	if !strings.Contains(history.Diff, "local change") {
		t.Fatalf("history diff missing latest file change:\n%s", history.Diff)
	}

	commit, err := c.Build(Command{Kind: KindCommit, Commit: "HEAD", Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("commit build failed: %v", err)
	}
	if !hasPath(commit.Files, "app.txt") || !strings.Contains(commit.Diff, "local change") {
		t.Fatalf("commit view incomplete: files=%#v diff=\n%s", commit.Files, commit.Diff)
	}

	appendFile(t, repo, "app.txt", "stash change\n")
	runGit(t, repo, "stash", "push", "-m", "test stash")
	stash, err := c.Build(Command{Kind: KindStash, Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("stash build failed: %v", err)
	}
	if len(stash.Stashes) != 1 || !strings.Contains(stash.Diff, "stash change") {
		t.Fatalf("stash view incomplete: stashes=%#v diff=\n%s", stash.Stashes, stash.Diff)
	}
}

func TestControllerAheadAndBehind(t *testing.T) {
	repo, remote := newRepoWithRemote(t)
	g := Git{Dir: repo}
	c := Controller{Git: g}

	appendFile(t, repo, "app.txt", "ahead local\n")
	runGit(t, repo, "add", "app.txt")
	runGit(t, repo, "commit", "-m", "ahead local")

	ahead, err := c.Build(Command{Kind: KindAhead, Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("ahead build failed: %v", err)
	}
	if !strings.Contains(ahead.Diff, "ahead local") {
		t.Fatalf("ahead diff missing local commit:\n%s", ahead.Diff)
	}

	runGit(t, repo, "reset", "--hard", "origin/feature")
	other := cloneRepo(t, remote)
	runGit(t, other, "checkout", "feature")
	appendFile(t, other, "app.txt", "behind remote\n")
	runGit(t, other, "add", "app.txt")
	runGit(t, other, "commit", "-m", "behind remote")
	runGit(t, other, "push", "origin", "feature")
	runGit(t, repo, "fetch", "origin")

	behind, err := c.Build(Command{Kind: KindBehind, Options: Options{Layout: LayoutSplit}})
	if err != nil {
		t.Fatalf("behind build failed: %v", err)
	}
	if !strings.Contains(behind.Diff, "behind remote") {
		t.Fatalf("behind diff missing remote commit:\n%s", behind.Diff)
	}
}

func TestTUIReloadAndResetShowsSelectedFileDiff(t *testing.T) {
	repo := newFixtureRepo(t)
	writeFile(t, repo, "alpha.txt", "alpha base\n")
	writeFile(t, repo, "beta.txt", "beta base\n")
	runGit(t, repo, "add", "alpha.txt", "beta.txt")
	runGit(t, repo, "commit", "-m", "add files")

	appendFile(t, repo, "alpha.txt", "alpha local\n")
	appendFile(t, repo, "beta.txt", "beta local\n")

	g := Git{Dir: repo}
	m := tuiModel{
		git:         g,
		controller:  Controller{Git: g},
		cmd:         Command{Kind: KindLocal, Options: Options{Layout: LayoutSplit}},
		layout:      LayoutSplit,
		width:       100,
		height:      30,
		showSidebar: true,
	}
	m.reloadAndReset()

	if len(m.view.Files) < 2 {
		t.Fatalf("files = %#v, want at least two changed files", m.view.Files)
	}
	selected := m.view.Files[0].Path
	if !strings.Contains(m.view.GitCommand, "-- "+selected) {
		t.Fatalf("git command = %q, want selected file %q", m.view.GitCommand, selected)
	}
	for _, file := range m.view.Files[1:] {
		if strings.Contains(m.view.Diff, " b/"+file.Path) || strings.Contains(m.view.Diff, " a/"+file.Path) {
			t.Fatalf("diff includes non-selected file %q after reload:\n%s", file.Path, m.view.Diff)
		}
	}
}

func TestTUIComparePickerCancelKeepsLocalFileNavigation(t *testing.T) {
	repo := newFixtureRepo(t)
	writeFile(t, repo, "alpha.txt", "alpha base\n")
	writeFile(t, repo, "beta.txt", "beta base\n")
	runGit(t, repo, "add", "alpha.txt", "beta.txt")
	runGit(t, repo, "commit", "-m", "add files")

	appendFile(t, repo, "alpha.txt", "alpha local\n")
	appendFile(t, repo, "beta.txt", "beta local\n")

	g := Git{Dir: repo}
	m := tuiModel{
		git:         g,
		controller:  Controller{Git: g},
		cmd:         Command{Kind: KindLocal, Options: Options{Layout: LayoutSplit}},
		layout:      LayoutSplit,
		width:       100,
		height:      30,
		showSidebar: true,
	}
	m.reloadAndReset()
	m.overlay = overlayModes
	m.setPickerItems([]string{"Local changes", "Compare/review branches"})
	m.pickerCursor = 1
	m.choosePicker()

	if m.overlay != overlayBranch {
		t.Fatalf("overlay = %v, want branch picker", m.overlay)
	}
	if m.cmd.Kind != KindCompare {
		t.Fatalf("pending command = %s, want compare", m.cmd.Kind)
	}

	updated, _ := m.updateOverlayKey("esc")
	m = updated.(tuiModel)
	if m.cmd.Kind != KindLocal {
		t.Fatalf("command after cancel = %s, want local", m.cmd.Kind)
	}

	m.moveCursor(1)
	selected := m.view.Files[m.cursor].Path
	if !strings.Contains(m.view.GitCommand, "-- "+selected) {
		t.Fatalf("git command after moving = %q, want selected file %q", m.view.GitCommand, selected)
	}
}

func TestTUIStashSelectionDoesNotResetCursor(t *testing.T) {
	repo := newFixtureRepo(t)

	appendFile(t, repo, "app.txt", "first stash change\n")
	runGit(t, repo, "stash", "push", "-m", "first stash")
	appendFile(t, repo, "app.txt", "second stash change\n")
	runGit(t, repo, "stash", "push", "-m", "second stash")

	g := Git{Dir: repo}
	m := tuiModel{
		git:         g,
		controller:  Controller{Git: g},
		cmd:         Command{Kind: KindStash, Options: Options{Layout: LayoutSplit}},
		layout:      LayoutSplit,
		width:       100,
		height:      30,
		showSidebar: true,
	}
	m.reloadAndReset()
	if len(m.view.Stashes) < 2 {
		t.Fatalf("stashes = %#v, want at least two", m.view.Stashes)
	}
	selectedRef := m.view.Stashes[1].Ref

	m.moveCursor(1)

	if m.cursor != 1 {
		t.Fatalf("cursor reset to %d, want 1", m.cursor)
	}
	if m.cmd.StashRef != selectedRef || m.view.Subtitle != selectedRef {
		t.Fatalf("selected stash = cmd %q subtitle %q, want %q", m.cmd.StashRef, m.view.Subtitle, selectedRef)
	}
	if !strings.Contains(m.view.Diff, "first stash change") {
		t.Fatalf("stash diff did not load selected stash:\n%s", m.view.Diff)
	}
}

func TestTUIFileCompareUsesFileSelectorBeforeBranch(t *testing.T) {
	repo := newFixtureRepo(t)
	g := Git{Dir: repo}
	m := tuiModel{
		git:         g,
		controller:  Controller{Git: g},
		cmd:         Command{Kind: KindFile, Options: Options{Layout: LayoutSplit}},
		layout:      LayoutSplit,
		width:       100,
		height:      30,
		showSidebar: true,
	}

	m.loadInitial()
	if m.overlay != overlayFile {
		t.Fatalf("overlay = %v, want file picker", m.overlay)
	}
	if !containsString(m.pickerItems, "app.txt") {
		t.Fatalf("file picker items = %#v, want app.txt", m.pickerItems)
	}

	m.choosePicker()
	if m.overlay != overlayBranch {
		t.Fatalf("overlay = %v, want branch picker", m.overlay)
	}
	if m.cmd.Path != "app.txt" {
		t.Fatalf("selected path = %q, want app.txt", m.cmd.Path)
	}
	if !containsString(m.pickerItems, "main") {
		t.Fatalf("branch picker items = %#v, want main", m.pickerItems)
	}

	updated, _ := m.updateOverlayKey("m")
	m = updated.(tuiModel)
	m.choosePicker()
	if m.overlay != overlayNone {
		t.Fatalf("overlay = %v, want none", m.overlay)
	}
	if m.cmd.Ref != "main" || m.view.Mode != KindFile {
		t.Fatalf("file compare command/view = %#v %#v", m.cmd, m.view)
	}
	if !strings.Contains(m.view.Diff, "feature line") {
		t.Fatalf("file compare diff missing expected content:\n%s", m.view.Diff)
	}
}

func TestTUIFileCompareWithPathUsesBranchSelector(t *testing.T) {
	repo := newFixtureRepo(t)
	g := Git{Dir: repo}
	m := tuiModel{
		git:         g,
		controller:  Controller{Git: g},
		cmd:         Command{Kind: KindFile, Path: "app.txt", Options: Options{Layout: LayoutSplit}},
		layout:      LayoutSplit,
		width:       100,
		height:      30,
		showSidebar: true,
	}

	m.loadInitial()
	if m.overlay != overlayBranch {
		t.Fatalf("overlay = %v, want branch picker", m.overlay)
	}
	if containsString(m.pickerItems, "app.txt") {
		t.Fatalf("path was already selected, should not show file picker items: %#v", m.pickerItems)
	}

	updated, _ := m.updateOverlayKey("m")
	m = updated.(tuiModel)
	m.choosePicker()
	if m.overlay != overlayNone {
		t.Fatalf("overlay = %v, want none", m.overlay)
	}
	if m.cmd.Ref != "main" || m.cmd.Path != "app.txt" || m.view.Mode != KindFile {
		t.Fatalf("file compare command/view = %#v %#v", m.cmd, m.view)
	}
}

func newFixtureRepo(t *testing.T) string {
	t.Helper()
	if err := os.MkdirAll(".testtmp", 0o755); err != nil {
		t.Fatal(err)
	}
	repo, err := os.MkdirTemp(".testtmp", "repo-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(repo)
	})

	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	writeFile(t, repo, "app.txt", "line one\n")
	runGit(t, repo, "add", "app.txt")
	runGit(t, repo, "commit", "-m", "initial")
	runGit(t, repo, "checkout", "-b", "feature")
	appendFile(t, repo, "app.txt", "feature line\n")
	runGit(t, repo, "add", "app.txt")
	runGit(t, repo, "commit", "-m", "feature")
	return repo
}

func newRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()
	repo := newFixtureRepo(t)
	remote, err := os.MkdirTemp(".testtmp", "remote-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(remote)
	})
	runGit(t, remote, "init", "--bare")
	runGit(t, repo, "remote", "add", "origin", absPath(t, remote))
	runGit(t, repo, "push", "-u", "origin", "main")
	runGit(t, repo, "push", "-u", "origin", "feature")
	return repo, remote
}

func cloneRepo(t *testing.T, remote string) string {
	t.Helper()
	dir, err := os.MkdirTemp(".testtmp", "clone-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	_ = os.RemoveAll(dir)
	cmd := exec.Command("git", "clone", absPath(t, remote), dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, out)
	}
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	return dir
}

func absPath(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, dir, path, contents string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, dir, path, contents string) {
	t.Helper()
	full := filepath.Join(dir, path)
	file, err := os.OpenFile(full, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(contents); err != nil {
		t.Fatal(err)
	}
}

func hasPath(files []FileStat, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
