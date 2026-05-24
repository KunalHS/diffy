package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	xansi "github.com/charmbracelet/x/ansi"
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

func TestDiffHeadersShowFileNameOnly(t *testing.T) {
	diff := `diff --git a/core/src/main/java/com/ontic/core/scheduler/jobs/watchconfig/handler/PublicRecordWatchConfigProcessor.java b/core/src/main/java/com/ontic/core/scheduler/jobs/watchconfig/handler/PublicRecordWatchConfigProcessor.java
--- a/core/src/main/java/com/ontic/core/scheduler/jobs/watchconfig/handler/PublicRecordWatchConfigProcessor.java
+++ b/core/src/main/java/com/ontic/core/scheduler/jobs/watchconfig/handler/PublicRecordWatchConfigProcessor.java
@@ -1,1 +1,1 @@
-old
+new`

	for _, layout := range []Layout{LayoutSplit, LayoutUnified} {
		joined := strings.Join(RenderDiff(diff, layout, 120), "\n")
		if strings.Contains(joined, "core/src/main/java") {
			t.Fatalf("%s header should not include full path:\n%s", layout, joined)
		}
		if !strings.Contains(joined, "PublicRecordWatchConfigProcessor.java") {
			t.Fatalf("%s header missing file name:\n%s", layout, joined)
		}
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

func TestSidebarToggleWritesState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	model := tuiModel{
		showSidebar: true,
		state:       DiffyState{Path: statePath},
	}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	toggled := updated.(tuiModel)
	if toggled.showSidebar {
		t.Fatalf("sidebar did not toggle closed")
	}

	state := LoadState(statePath)
	if state.Sidebar || !state.SidebarSet {
		t.Fatalf("state sidebar = %t set=%t, want false set", state.Sidebar, state.SidebarSet)
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

func TestModeShortcutIsM(t *testing.T) {
	model := tuiModel{cmd: Command{Kind: KindLocal}}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	withModes := updated.(tuiModel)
	if withModes.overlay != overlayModes {
		t.Fatalf("m key overlay = %v, want mode selector", withModes.overlay)
	}

	updated, _ = model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	withoutModes := updated.(tuiModel)
	if withoutModes.overlay == overlayModes {
		t.Fatalf("/ key should not open mode selector")
	}

	bottom := model.renderBottom()
	if !strings.Contains(bottom, "m modes") || strings.Contains(bottom, "/ modes") {
		t.Fatalf("bottom hint = %q, want m modes only", bottom)
	}
}

func TestSlashSearchesSidebar(t *testing.T) {
	model := tuiModel{
		cmd: Command{Kind: KindFile},
		view: ViewData{
			Files: []FileStat{
				{Path: "src/App.java", Add: 1},
				{Path: "src/Thing.java", Add: 2},
			},
			Diff: "existing diff",
		},
	}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	searching := updated.(tuiModel)
	if searching.overlay != overlayNone || !searching.sidebarEditing {
		t.Fatalf("slash state = overlay %v editing %t, want inline sidebar search", searching.overlay, searching.sidebarEditing)
	}

	for _, r := range "thing" {
		updated, _ = searching.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		searching = updated.(tuiModel)
	}
	if got := searching.sidebarDisplayIndices(len(searching.view.Files)); len(got) != 1 || got[0] != 1 {
		t.Fatalf("live sidebar filter = %#v, want only row 1", got)
	}
	if searching.cursor != 1 {
		t.Fatalf("sidebar search cursor = %d, want matching row 1", searching.cursor)
	}
	if searching.diffFocused {
		t.Fatalf("sidebar search should leave focus on sidebar")
	}

	sidebar := searching.renderSidebar(40, 8)
	if !strings.Contains(sidebar, "/ thing") || strings.Contains(sidebar, "/ files:") || strings.Contains(sidebar, "App.java") {
		t.Fatalf("sidebar search render did not show locked filter state:\n%s", sidebar)
	}

	updated, _ = searching.updateKey(tea.KeyMsg{Type: tea.KeyEnter})
	locked := updated.(tuiModel)
	if locked.sidebarEditing || locked.sidebarSearch != "thing" {
		t.Fatalf("locked sidebar search = editing %t query %q, want locked query", locked.sidebarEditing, locked.sidebarSearch)
	}

	updated, _ = locked.updateKey(tea.KeyMsg{Type: tea.KeyEsc})
	cleared := updated.(tuiModel)
	if cleared.sidebarSearchVisible() || len(cleared.sidebarDisplayIndices(len(cleared.view.Files))) != 2 {
		t.Fatalf("esc did not clear sidebar search: query=%q editing=%t", cleared.sidebarSearch, cleared.sidebarEditing)
	}
}

func TestSidebarSearchUsesVisibleFileNameNotHiddenPath(t *testing.T) {
	model := tuiModel{
		cmd: Command{Kind: KindLocal},
		view: ViewData{
			Files: []FileStat{
				{Path: "core/src/main/java/com/ontic/core/topic/common/TopicType.java", Add: 1},
				{Path: "core/src/main/java/com/ontic/core/watch/WatchConfigListeningJob.java", Add: 1},
			},
		},
	}

	model.sidebarSearch = "core"
	got := model.sidebarDisplayIndices(len(model.view.Files))
	if len(got) != 0 {
		t.Fatalf("sidebar search matched hidden path segments: %#v", got)
	}

	model.sidebarSearch = "topic"
	got = model.sidebarDisplayIndices(len(model.view.Files))
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("sidebar search should match visible file name text, got %#v", got)
	}
}

func TestSlashSearchesRenderedDiffAndNMovesMatches(t *testing.T) {
	diff := `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,5 +1,5 @@
 alpha
-old one
+needle one
 middle
-old two
+needle two
 omega`
	model := tuiModel{
		width:       100,
		layout:      LayoutSplit,
		diffFocused: true,
		cmd:         Command{Kind: KindFile},
		view:        ViewData{Diff: diff},
	}
	model.diffSearch = "needle"
	matches := model.diffSearchOccurrences()
	if len(matches) != 2 {
		t.Fatalf("diff search matches = %#v, want two rendered matches", matches)
	}
	model.diffSearch = ""

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	searching := updated.(tuiModel)
	if searching.overlay != overlayNone || !searching.diffEditing {
		t.Fatalf("slash state = overlay %v editing %t, want inline diff search", searching.overlay, searching.diffEditing)
	}

	for _, r := range "needle" {
		updated, _ = searching.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		searching = updated.(tuiModel)
	}
	main := searching.renderMain(100, 12)
	if !strings.Contains(main, "/ needle") || strings.Contains(main, "/ diff:") {
		t.Fatalf("diff search render missing inline search box:\n%s", main)
	}

	updated, _ = searching.updateKey(tea.KeyMsg{Type: tea.KeyEnter})
	locked := updated.(tuiModel)
	if locked.diffEditing || locked.diffCurrent != 0 || locked.scroll != matches[0].Line || locked.hScroll != 0 || !locked.diffFocused {
		t.Fatalf("diff search position = editing %t current %d scroll %d h %d focused %t, want first match line %d and focused", locked.diffEditing, locked.diffCurrent, locked.scroll, locked.hScroll, locked.diffFocused, matches[0].Line)
	}

	updated, _ = locked.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	next := updated.(tuiModel)
	if next.diffCurrent != 1 || next.scroll != matches[1].Line {
		t.Fatalf("n search = current %d scroll %d, want second match line %d", next.diffCurrent, next.scroll, matches[1].Line)
	}

	updated, _ = next.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	prev := updated.(tuiModel)
	if prev.diffCurrent != 0 || prev.scroll != matches[0].Line {
		t.Fatalf("N search = current %d scroll %d, want first match line %d", prev.diffCurrent, prev.scroll, matches[0].Line)
	}

	updated, _ = prev.updateKey(tea.KeyMsg{Type: tea.KeyEsc})
	cleared := updated.(tuiModel)
	if cleared.diffSearchVisible() || cleared.diffCurrent != -1 {
		t.Fatalf("esc did not clear diff search: query=%q editing=%t current=%d", cleared.diffSearch, cleared.diffEditing, cleared.diffCurrent)
	}
	if cleared.message != "" {
		t.Fatalf("clearing diff search should not add footer message, got %q", cleared.message)
	}
}

func TestSearchHighlightStylesAreDistinct(t *testing.T) {
	if fmt.Sprint(styleSearchMatch.GetBackground()) == fmt.Sprint(styleSearchCurrent.GetBackground()) {
		t.Fatalf("current search match should use a different background from other matches")
	}
}

func TestInlineSearchLinesAreCompact(t *testing.T) {
	model := tuiModel{
		cmd:            Command{Kind: KindLocal},
		sidebarSearch:  "topic",
		sidebarEditing: true,
		diffSearch:     "override",
		diffEditing:    true,
		view: ViewData{Diff: `diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,1 +1,1 @@
-old
+override`},
	}

	sidebarLine := model.renderSidebarSearchLine(40)
	if strings.Contains(sidebarLine, "\n") || strings.Contains(sidebarLine, "files:") {
		t.Fatalf("sidebar search line should be one compact unlabeled row: %q", sidebarLine)
	}

	diffLine := model.renderDiffSearchLine(60)
	diffText := strings.TrimSpace(xansi.Strip(diffLine))
	if strings.Contains(diffLine, "\n") || strings.Contains(diffLine, "diff:") {
		t.Fatalf("diff search line should be one compact unlabeled row: %q", diffLine)
	}
	if !strings.HasPrefix(diffText, "/ override|") || !strings.HasSuffix(diffText, "1/1") {
		t.Fatalf("diff search line should keep query and count on one row, got %q", diffText)
	}
}

func TestSidebarSelectedSearchKeepsSelectionAndHighlight(t *testing.T) {
	model := tuiModel{
		cmd: Command{Kind: KindLocal},
		view: ViewData{
			Files: []FileStat{
				{Path: "WatchConfigListeningJob.java", Add: 17, Del: 1},
				{Path: "PublicRecordWatchConfigProcessor.java", Add: 19, Del: 2},
			},
		},
		cursor:        1,
		sidebarSearch: "watch",
	}

	sidebar := model.renderSidebar(44, 8)
	if !strings.Contains(xansi.Strip(sidebar), "PublicRecordWatchConfigPr") {
		t.Fatalf("sidebar lost selected row text:\n%s", sidebar)
	}
	if !strings.Contains(sidebar, styleSearchMatch.Render("Watch")) {
		t.Fatalf("sidebar did not highlight selected row match:\n%s", sidebar)
	}
}

func TestSearchClearDoesNotPolluteFooter(t *testing.T) {
	model := tuiModel{
		cmd:           Command{Kind: KindLocal},
		sidebarSearch: "watch",
	}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyEsc})
	cleared := updated.(tuiModel)
	if cleared.message != "" {
		t.Fatalf("clearing sidebar search should not add footer message, got %q", cleared.message)
	}
	if strings.Contains(cleared.renderBottom(), "search cleared") {
		t.Fatalf("footer should not include search cleared message: %q", cleared.renderBottom())
	}
}

func TestDiffSearchIgnoresCollapsedContextLabels(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`diff --git a/app.txt b/app.txt
--- a/app.txt
+++ b/app.txt
@@ -1,14 +1,14 @@
-before
+after
`)
	for i := 0; i < 10; i++ {
		builder.WriteString(" same content\n")
	}
	builder.WriteString(`-old end
+new end`)

	model := tuiModel{
		width:       120,
		height:      24,
		cmd:         Command{Kind: KindLocal},
		layout:      LayoutSplit,
		diffFocused: true,
		diffSearch:  "lines",
		view:        ViewData{Diff: builder.String()},
	}

	rendered := model.renderMain(120, 18)
	if !strings.Contains(xansi.Strip(rendered), "unchanged lines") {
		t.Fatalf("test setup should render collapsed context label:\n%s", rendered)
	}
	if matches := model.diffSearchOccurrences(); len(matches) != 0 {
		t.Fatalf("diff search should ignore collapsed context labels, got %#v", matches)
	}
	if !strings.Contains(xansi.Strip(rendered), "/ lines") || !strings.Contains(xansi.Strip(rendered), "0/0") {
		t.Fatalf("search status should show the query with no matches:\n%s", rendered)
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

func TestLayoutToggleWritesState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	model := tuiModel{
		layout: LayoutSplit,
		state:  DiffyState{Path: statePath},
	}

	updated, _ := model.updateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	unified := updated.(tuiModel)
	if unified.layout != LayoutUnified {
		t.Fatalf("layout = %q, want unified", unified.layout)
	}

	state := LoadState(statePath)
	if state.Layout != LayoutUnified || !state.LayoutSet {
		t.Fatalf("state layout = %q set=%t, want unified set", state.Layout, state.LayoutSet)
	}
}

func TestModeOverlayQQuits(t *testing.T) {
	model := tuiModel{overlay: overlayModes}

	_, cmd := model.updateOverlayKey("q")
	if cmd == nil {
		t.Fatal("q in mode selector did not return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("q in mode selector returned %T, want tea.QuitMsg", msg)
	}
}

func TestPickerOverlayFiltersTypedInput(t *testing.T) {
	model := tuiModel{
		overlay:     overlayBranch,
		pickerTitle: "Choose target branch",
	}
	model.setPickerItems([]string{"main", "use-1234-qa", "feature/kumar-debug"})

	updated, _ := model.updateOverlayKey("u")
	filtered := updated.(tuiModel)
	if filtered.pickerFilter != "u" {
		t.Fatalf("filter = %q, want u", filtered.pickerFilter)
	}
	if len(filtered.pickerItems) != 2 {
		t.Fatalf("filtered items = %#v, want two u matches", filtered.pickerItems)
	}

	updated, _ = filtered.updateOverlayKey("k")
	filtered = updated.(tuiModel)
	if filtered.pickerFilter != "uk" {
		t.Fatalf("k should type into filter, got filter %q", filtered.pickerFilter)
	}

	updated, _ = filtered.updateOverlayKey("backspace")
	filtered = updated.(tuiModel)
	if filtered.pickerFilter != "u" {
		t.Fatalf("backspace filter = %q, want u", filtered.pickerFilter)
	}
}

func TestPickerOverlayPinsFilterWhenResultsOverflow(t *testing.T) {
	model := tuiModel{
		overlay:     overlayBranch,
		pickerTitle: "Choose target branch",
		width:       120,
		height:      20,
	}
	items := make([]string, 40)
	for i := range items {
		items[i] = fmt.Sprintf("branch-%02d", i)
	}
	model.setPickerItems(items)
	for i := 0; i < 30; i++ {
		updated, _ := model.updateOverlayKey("down")
		model = updated.(tuiModel)
	}

	overlay := model.renderOverlay()
	if !strings.Contains(overlay, "Choose target branch") || !strings.Contains(overlay, "filter: ") {
		t.Fatalf("picker header/filter not pinned:\n%s", overlay)
	}
	if lines := strings.Count(overlay, "\n") + 1; lines > model.height {
		t.Fatalf("picker height = %d, want <= terminal height %d:\n%s", lines, model.height, overlay)
	}
	if !strings.Contains(overlay, "branch-30") {
		t.Fatalf("picker did not scroll to selected result:\n%s", overlay)
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

func TestCompareSidebarUsesSameFileListAsLocal(t *testing.T) {
	model := tuiModel{
		cmd: Command{Kind: KindCompare},
		view: ViewData{
			Files: []FileStat{
				{Path: "src/App.java", Add: 3, Del: 1},
				{Path: "src/Thing.java", Add: 2},
			},
			Commits: []CommitItem{{Hash: "abc123", Raw: "abc123 compare commit"}},
		},
	}

	sidebar := model.renderSidebar(40, 12)
	if !strings.Contains(sidebar, "Files") || !strings.Contains(sidebar, "App.java +3 -1") {
		t.Fatalf("compare sidebar missing file list:\n%s", sidebar)
	}
	if strings.Contains(sidebar, "Commits") || strings.Contains(sidebar, "compare commit") {
		t.Fatalf("compare sidebar should stay files-only like local:\n%s", sidebar)
	}
}

func TestSidebarSelectionDimsWhenDiffFocused(t *testing.T) {
	model := tuiModel{
		cmd: Command{Kind: KindLocal},
		view: ViewData{
			Files: []FileStat{
				{Path: "src/App.java", Add: 3, Del: 1},
				{Path: "src/Thing.java", Add: 2},
			},
		},
		cursor: 1,
	}

	activeStyle := sidebarSelectionStyle(false)
	dimmedStyle := sidebarSelectionStyle(true)
	if fmt.Sprint(activeStyle.GetBackground()) == fmt.Sprint(dimmedStyle.GetBackground()) {
		t.Fatalf("focused and unfocused sidebar selections use the same background")
	}
	if fmt.Sprint(activeStyle.GetForeground()) == fmt.Sprint(dimmedStyle.GetForeground()) {
		t.Fatalf("focused and unfocused sidebar selections use the same foreground")
	}

	model.diffFocused = true
	dimmed := model.renderSidebar(40, 8)
	if !strings.Contains(dimmed, "Thing.java +2 -0") {
		t.Fatalf("dimmed sidebar lost selected file text:\n%s", dimmed)
	}
}

func TestStashSidebarShowsBoundedStashListOnly(t *testing.T) {
	stashes := make([]StashItem, 40)
	for i := range stashes {
		stashes[i] = StashItem{
			Ref:     fmt.Sprintf("stash@{%d}", i),
			Subject: fmt.Sprintf("WIP on branch-%02d with a long subject", i),
		}
	}
	model := tuiModel{
		cmd:    Command{Kind: KindStash},
		view:   ViewData{Stashes: stashes, Files: []FileStat{{Path: "app.go", Add: 1}}},
		cursor: 35,
	}

	sidebar := model.renderSidebar(48, 12)
	if !strings.Contains(sidebar, "Stashes") || !strings.Contains(sidebar, "stash@{35}") {
		t.Fatalf("stash sidebar did not keep selected stash visible:\n%s", sidebar)
	}
	if strings.Contains(sidebar, "Files") || strings.Contains(sidebar, "app.go") {
		t.Fatalf("stash sidebar should not mix files into the selectable stash list:\n%s", sidebar)
	}
	if strings.Contains(sidebar, "stash@{0}") {
		t.Fatalf("stash sidebar did not scroll to selected stash:\n%s", sidebar)
	}
}

func TestMouseSelectUsesSameScrollResetAsKeyboard(t *testing.T) {
	model := tuiModel{
		width:       80,
		height:      20,
		showSidebar: true,
		cmd:         Command{Kind: KindFile},
		view: ViewData{
			Files: []FileStat{{Path: "one.go"}, {Path: "two.go"}},
			Diff:  "old diff",
		},
		scroll:  12,
		hScroll: 8,
	}

	updated := model.updateMouse(tea.MouseMsg{
		Type:   tea.MouseLeft,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      2,
		Y:      3,
	})

	if updated.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", updated.cursor)
	}
	if updated.scroll != 0 || updated.hScroll != 0 {
		t.Fatalf("scroll reset = (%d,%d), want (0,0)", updated.scroll, updated.hScroll)
	}
}

func TestMouseSelectUsesVisibleSidebarOffset(t *testing.T) {
	files := make([]FileStat, 40)
	for i := range files {
		files[i] = FileStat{Path: fmt.Sprintf("file-%02d.go", i)}
	}
	model := tuiModel{
		width:       90,
		height:      14,
		showSidebar: true,
		cmd:         Command{Kind: KindFile},
		view:        ViewData{Files: files, Diff: "old diff"},
		cursor:      35,
	}

	updated := model.updateMouse(tea.MouseMsg{
		Type:   tea.MouseLeft,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      2,
		Y:      2,
	})

	if updated.cursor != 25 {
		t.Fatalf("cursor = %d, want visible row offset 25", updated.cursor)
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
