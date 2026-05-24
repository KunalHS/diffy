package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSplitRenderUsesFullWidthForNewFileAdditions(t *testing.T) {
	diff := `diff --git a/new.txt b/new.txt
new file mode 100644
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,2 @@
+line one
+line two`

	lines := RenderDiff(diff, LayoutSplit, 80)
	for _, line := range lines {
		if strings.Contains(line, "line one") && strings.Contains(line, "|") {
			t.Fatalf("new-file addition rendered in side-by-side columns: %q", line)
		}
	}
}

func TestSplitRenderHidesPatchHeaders(t *testing.T) {
	diff := `diff --git a/COMMANDS.md b/COMMANDS.md
new file mode 100644
index 0000000..4a77c9a
--- /dev/null
+++ b/COMMANDS.md
@@ -0,0 +1,1 @@
+# Diffy Commands`

	lines := RenderDiff(diff, LayoutSplit, 80)
	joined := strings.Join(lines, "\n")
	for _, hidden := range []string{"diff --git", "new file mode", "index 0000000", "--- /dev/null", "+++ b/COMMANDS.md", "@@ -0,0"} {
		if strings.Contains(joined, hidden) {
			t.Fatalf("split render still shows %q:\n%s", hidden, joined)
		}
	}
	for _, line := range lines {
		if (strings.Contains(line, "--- /dev/null") || strings.Contains(line, "+++ b/COMMANDS.md")) && strings.Contains(line, "|") {
			t.Fatalf("file header rendered in split columns: %q", line)
		}
	}
}

func TestRenderHidesNoisyDiffHeaders(t *testing.T) {
	diff := `diff --git a/PLAN.md b/PLAN.md
new file mode 100644
index 0000000..f738756
--- /dev/null
+++ b/PLAN.md
@@ -0,0 +1,1 @@
+# Diffy Plan`

	for _, layout := range []Layout{LayoutSplit, LayoutUnified} {
		lines := RenderDiff(diff, layout, 100)
		joined := strings.Join(lines, "\n")
		for _, hidden := range []string{"diff --git", "new file mode", "index 0000000", "--- /dev/null", "+++ b/PLAN.md", "@@ -0,0"} {
			if strings.Contains(joined, hidden) {
				t.Fatalf("%s render still shows %q:\n%s", layout, hidden, joined)
			}
		}
		if !strings.Contains(joined, "# Diffy Plan") {
			t.Fatalf("%s render removed actual content:\n%s", layout, joined)
		}
	}
}

func TestSplitRenderUsesPaneDividerNotTextPipe(t *testing.T) {
	diff := `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,2 +1,2 @@
 same
-old
+new`

	lines := RenderDiff(diff, LayoutSplit, 100)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, " | ") {
		t.Fatalf("split render uses text pipe divider:\n%s", joined)
	}
	if !strings.Contains(joined, "│") {
		t.Fatalf("split render missing pane boundary:\n%s", joined)
	}
}

func TestSplitRenderShowsChangeLaneMarkers(t *testing.T) {
	diff := `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,3 +1,3 @@
-old one
-old two
-old three
+new one
+new two
+new three`

	lines := RenderDiff(diff, LayoutSplit, 120)
	joined := strings.Join(lines, "\n")
	for _, marker := range []string{"╭", "┃", "╰"} {
		if !strings.Contains(joined, marker) {
			t.Fatalf("split render missing change lane marker %q:\n%s", marker, joined)
		}
	}
}

func TestModifiedRowsRenderPairedOldAndNewTokens(t *testing.T) {
	diff := `diff --git a/App.java b/App.java
--- a/App.java
+++ b/App.java
@@ -32,1 +32,1 @@
-import com.ontic.fwk.common.IdLabel;
+import com.ontic.core.signals.topic.TopicUtils;`

	for _, layout := range []Layout{LayoutSplit, LayoutUnified} {
		lines := RenderDiff(diff, layout, 140)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "IdLabel") || !strings.Contains(joined, "TopicUtils") {
			t.Fatalf("%s render missing modified tokens:\n%s", layout, joined)
		}
	}
}

func TestUnifiedRenderUsesLineNumberGutters(t *testing.T) {
	diff := `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -7,2 +7,2 @@
 same
-old
+new`

	lines := RenderDiff(diff, LayoutUnified, 80)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"  7   7", "  8     - old", "      8 + new"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("unified render missing gutter/content %q:\n%s", want, joined)
		}
	}
}

func TestRenderCollapsesAndExpandsContext(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,14 +1,14 @@
-before
+after
`)
	for i := 0; i < 10; i++ {
		builder.WriteString(" same line\n")
	}
	builder.WriteString(`-old end
+new end`)

	diff := builder.String()
	collapsed := strings.Join(RenderDiffViewportWithOptions(diff, LayoutUnified, 100, 0, DiffRenderOptions{}), "\n")
	if !strings.Contains(collapsed, "... ") || !strings.Contains(collapsed, "unchanged lines") {
		t.Fatalf("large context did not collapse:\n%s", collapsed)
	}

	expanded := strings.Join(RenderDiffViewportWithOptions(diff, LayoutUnified, 100, 0, DiffRenderOptions{ExpandAll: true}), "\n")
	if strings.Contains(expanded, "unchanged lines") {
		t.Fatalf("expanded render still has collapsed context:\n%s", expanded)
	}
}

func TestRenderKeepsPatchLikeContentInsideHunks(t *testing.T) {
	diff := `diff --git a/text.txt b/text.txt
index 1111111..2222222 100644
--- a/text.txt
+++ b/text.txt
@@ -1,1 +1,4 @@
 line one
+diff --git is content here
+index 0000000 is content here
+--- /dev/null is content here`

	lines := RenderDiff(diff, LayoutUnified, 100)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"diff --git is content here", "index 0000000 is content here", "--- /dev/null is content here"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("render dropped content %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "diff --git a/text.txt") || strings.Contains(joined, "index 1111111") {
		t.Fatalf("render kept patch metadata:\n%s", joined)
	}
}

func TestBottomHintShowsEscClearFileWhenRestricted(t *testing.T) {
	model := tuiModel{
		width: 120,
		cmd:   Command{Kind: KindLocal},
		view:  ViewData{RestrictedPath: "app.txt"},
	}
	bottom := model.renderBottom()
	if !strings.Contains(bottom, "esc clear file") {
		t.Fatalf("bottom hint = %q, want esc clear file hint", bottom)
	}
	if strings.Contains(bottom, "f clear file") {
		t.Fatalf("bottom hint contains removed f clear file shortcut: %q", bottom)
	}
}

func TestSidebarToggleExpandsMainPane(t *testing.T) {
	model := tuiModel{
		width:       80,
		height:      20,
		showSidebar: false,
		layout:      LayoutUnified,
		cmd:         Command{Kind: KindLocal},
		view: ViewData{
			GitCommand: "git diff",
			Diff:       "+hello",
		},
	}
	body := model.renderBody()
	if strings.Contains(body, "Files") {
		t.Fatalf("body contains sidebar while sidebar hidden:\n%s", body)
	}
}

func TestBottomHintShowsSidebarShortcutAsS(t *testing.T) {
	model := tuiModel{
		width: 120,
		cmd:   Command{Kind: KindLocal},
	}
	bottom := model.renderBottom()
	if !strings.Contains(bottom, "s sidebar") {
		t.Fatalf("bottom hint = %q, want s sidebar", bottom)
	}
	if strings.Contains(bottom, "u unstaged") || strings.Contains(bottom, "s staged") || strings.Contains(bottom, "tab sidebar") {
		t.Fatalf("bottom hint contains removed shortcuts: %q", bottom)
	}
}

func TestGitCommandHiddenFromMainByDefault(t *testing.T) {
	model := tuiModel{
		width:       80,
		height:      20,
		showSidebar: true,
		layout:      LayoutUnified,
		cmd:         Command{Kind: KindLocal},
		view: ViewData{
			GitCommand: "git diff -- app.txt",
			Diff:       "+hello",
		},
	}
	main := model.renderMain(60, 10)
	if strings.Contains(main, "git diff -- app.txt") {
		t.Fatalf("main pane shows git command by default:\n%s", main)
	}
}

func TestGitCommandShownAtBottomOfSidebarWhenToggled(t *testing.T) {
	model := tuiModel{
		width:       80,
		height:      20,
		showSidebar: true,
		showGitCmd:  true,
		cmd:         Command{Kind: KindLocal},
		view: ViewData{
			GitCommand: "git diff -- app.txt",
			Files:      []FileStat{{Path: "app.txt", Add: 1}},
		},
	}
	sidebar := model.renderSidebar(30, 8)
	if !strings.Contains(sidebar, "Git") || !strings.Contains(sidebar, "git diff -- app.txt") {
		t.Fatalf("sidebar missing toggled git command:\n%s", sidebar)
	}
}

func TestSidebarFileLineShowsFileNameOnly(t *testing.T) {
	line := renderFileLine(FileStat{
		Path: "core/src/main/java/com/ontic/core/topic/common/TopicType.java",
		Add:  12,
		Del:  3,
	}, 80)

	if strings.Contains(line, "core/src/main/java") {
		t.Fatalf("sidebar file line should not include full path: %q", line)
	}
	if !strings.Contains(line, "TopicType.java +12 -3") {
		t.Fatalf("sidebar file line = %q, want file name with counts", line)
	}
}

func TestEnterFocusesDiffAndLScrolls(t *testing.T) {
	model := tuiModel{
		width:  120,
		height: 30,
		cmd:    Command{Kind: KindLocal},
		view: ViewData{
			Files: []FileStat{{Path: "app.txt", Add: 1}},
			Diff: `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,1 +1,1 @@
-short
+this is a very long replacement line that should scroll horizontally`,
		},
	}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyEnter})
	focused := updated.(tuiModel)
	if !focused.diffFocused {
		t.Fatalf("enter did not focus diff pane")
	}

	updated, _ = focused.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	scrolled := updated.(tuiModel)
	if scrolled.hScroll != diffHorizontalScrollStep {
		t.Fatalf("l key hScroll = %d, want %d", scrolled.hScroll, diffHorizontalScrollStep)
	}
	updated, _ = scrolled.updateKey(tea.KeyMsg{Type: tea.KeyLeft})
	leftScrolled := updated.(tuiModel)
	if leftScrolled.hScroll >= scrolled.hScroll {
		t.Fatalf("left arrow did not reduce horizontal scroll: before=%d after=%d", scrolled.hScroll, leftScrolled.hScroll)
	}
	updated, _ = leftScrolled.updateKey(tea.KeyMsg{Type: tea.KeyRight})
	rightScrolled := updated.(tuiModel)
	if rightScrolled.hScroll <= leftScrolled.hScroll {
		t.Fatalf("right arrow did not advance horizontal scroll: before=%d after=%d", leftScrolled.hScroll, rightScrolled.hScroll)
	}
	bottom := scrolled.renderBottom()
	if !strings.Contains(bottom, "h/l scroll") || strings.Contains(bottom, "left/right scroll") {
		t.Fatalf("diff-focused bottom hint = %q, want h/l scroll only", bottom)
	}
}

func TestDiffFocusedJKScrollsByLineStep(t *testing.T) {
	model := tuiModel{diffFocused: true}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	down := updated.(tuiModel)
	if down.scroll != diffLineScrollStep {
		t.Fatalf("j key scroll = %d, want %d", down.scroll, diffLineScrollStep)
	}

	updated, _ = down.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	up := updated.(tuiModel)
	if up.scroll != 0 {
		t.Fatalf("k key should clamp back to 0, got %d", up.scroll)
	}
}

func TestZAndShiftZToggleCollapsedContext(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,14 +1,14 @@
-before
+after
`)
	for i := 0; i < 10; i++ {
		builder.WriteString(" same line\n")
	}
	builder.WriteString(`-old end
+new end`)

	model := tuiModel{
		width:       120,
		height:      30,
		cmd:         Command{Kind: KindLocal},
		layout:      LayoutUnified,
		diffFocused: true,
		view:        ViewData{Diff: builder.String()},
	}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	expanded := updated.(tuiModel)
	if len(expanded.expandedDiff) == 0 {
		t.Fatalf("z did not expand a collapsed context")
	}

	updated, _ = expanded.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	collapsed := updated.(tuiModel)
	if len(collapsed.expandedDiff) != 0 {
		t.Fatalf("second z did not collapse nearest context: %#v", collapsed.expandedDiff)
	}

	updated, _ = collapsed.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	all := updated.(tuiModel)
	if !all.expandAllDiff {
		t.Fatalf("Z did not toggle expand-all context")
	}

	updated, _ = all.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	none := updated.(tuiModel)
	if none.expandAllDiff {
		t.Fatalf("second Z did not collapse all context")
	}
}

func TestDiffViewportClipsInsteadOfTildeTruncating(t *testing.T) {
	diff := `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,1 +1,1 @@
-short
+abcdefghijklmnopqrstuvwxyz0123456789`

	lines := RenderDiffViewport(diff, LayoutUnified, 24, 0)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "~") {
		t.Fatalf("diff viewport should clip without tilde truncation:\n%s", joined)
	}

	scrolled := strings.Join(RenderDiffViewport(diff, LayoutUnified, 24, 10), "\n")
	if !strings.Contains(scrolled, "klmnopqrst") {
		t.Fatalf("horizontal viewport did not expose later content:\n%s", scrolled)
	}
}
