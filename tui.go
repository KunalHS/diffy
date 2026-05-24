package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayModes
	overlayBranch
	overlayFile
	overlayPrompt
	overlayConfirm
	overlayChangedFiles
)

type promptKind string

const (
	promptCompareTarget promptKind = "compare-target"
	promptRecentCount   promptKind = "recent-count"
	promptFileCompare   promptKind = "file-compare"
	promptHistoryFile   promptKind = "history-file"
	promptCommit        promptKind = "commit"
)

const (
	diffLineScrollStep       = 3
	diffHorizontalScrollStep = 16
	pageScrollStep           = 10
)

type tuiModel struct {
	cfg        Config
	state      DiffyState
	git        Git
	controller Controller
	cmd        Command
	view       ViewData

	width          int
	height         int
	cursor         int
	stashCursor    int
	commitCursor   int
	scroll         int
	hScroll        int
	layout         Layout
	overlay        overlayKind
	pickerTitle    string
	pickerAllItems []string
	pickerItems    []string
	pickerFilter   string
	pickerCursor   int
	pickerScroll   int
	promptKind     promptKind
	promptTitle    string
	input          string
	confirmText    string
	confirmCmd     []string
	message        string
	err            string
	lastHistory    *Command
	showSidebar    bool
	showGitCmd     bool
	diffFocused    bool
	expandedDiff   map[string]bool
	expandAllDiff  bool
}

func RunTUI(cmd Command, cfg Config, state DiffyState) error {
	m := newTUIModel(cmd, cfg, state)
	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func newTUIModel(cmd Command, cfg Config, state DiffyState) tuiModel {
	m := tuiModel{
		cfg:         cfg,
		state:       state,
		git:         Git{},
		controller:  Controller{Git: Git{}},
		cmd:         cmd,
		layout:      cmd.Options.Layout,
		width:       100,
		height:      30,
		showSidebar: cfg.Sidebar,
	}
	m.loadInitial()
	return m
}

func (m *tuiModel) loadInitial() {
	if m.cmd.Kind == KindCompare && m.cmd.Target == "" {
		m.openBranchPicker("Choose target branch")
		return
	}
	if (m.cmd.Options.FileFlag && m.cmd.Options.FilePath == "") && supportsFilePicker(m.cmd.Kind) {
		if err := m.reload(); err != nil {
			m.err = err.Error()
			return
		}
		m.openFilePicker("Restrict to file", filePaths(m.view.Files))
		return
	}
	if m.cmd.Kind == KindHistory && m.cmd.Path == "" {
		files, err := m.git.TrackedFiles()
		if err != nil {
			m.err = err.Error()
			return
		}
		m.openFilePicker("Choose file for history", files)
		return
	}
	if err := m.reload(); err != nil {
		m.err = err.Error()
	}
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg), nil
	}
	return m, nil
}

func (m tuiModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.overlay != overlayNone {
		return m.updateOverlayKey(key)
	}
	switch key {
	case "q":
		return m, tea.Quit
	case "esc":
		if m.diffFocused {
			m.diffFocused = false
			return m, nil
		}
		if m.canClearFileRestriction() {
			m.clearFileRestriction()
			return m, nil
		}
		if m.lastHistory != nil && m.cmd.Kind == KindCommit {
			m.cmd = *m.lastHistory
			m.lastHistory = nil
			m.reloadAndReset()
		}
	case "up", "k":
		if m.diffFocused {
			m.scroll -= diffLineScrollStep
			if m.scroll < 0 {
				m.scroll = 0
			}
		} else {
			m.moveCursor(-1)
		}
	case "down", "j":
		if m.diffFocused {
			m.scroll += diffLineScrollStep
		} else {
			m.moveCursor(1)
		}
	case "left", "h":
		if m.diffFocused {
			m.hScroll -= diffHorizontalScrollStep
			if m.hScroll < 0 {
				m.hScroll = 0
			}
		}
	case "l":
		if m.diffFocused {
			m.hScroll += diffHorizontalScrollStep
		} else if m.view.SourceIsCurrent && m.cmd.Kind == KindCompare {
			m.cmd = Command{Kind: KindLocal, Options: Options{Layout: m.layout}}
			m.reloadAndReset()
		}
	case "right":
		if m.diffFocused {
			m.hScroll += diffHorizontalScrollStep
		}
	case "z":
		if m.diffFocused {
			m.toggleNearestCollapsedContext()
		}
	case "Z":
		if m.diffFocused {
			m.toggleAllCollapsedContext()
		}
	case "pgup", "b":
		m.scroll -= pageScrollStep
		if m.scroll < 0 {
			m.scroll = 0
		}
	case "pgdown", " ":
		m.scroll += pageScrollStep
	case "f":
		if m.cmd.Kind == KindHistory {
			m.openChangedFilesForSelectedCommit()
		} else if !m.canClearFileRestriction() {
			m.restrictToFile()
		}
	case "enter":
		if m.diffFocused {
			m.diffFocused = false
		} else {
			m.focusDiff()
		}
	case "s":
		m.showSidebar = !m.showSidebar
		m.rememberSidebar()
	case "g":
		m.showGitCmd = !m.showGitCmd
	case "/":
		m.overlay = overlayModes
		m.pickerTitle = "Modes"
		m.setPickerItems([]string{"Local changes", "Compare/review branches", "Ahead of upstream", "Behind upstream", "Recent commits", "File compare", "Stash", "File history", "Commit view"})
	case "1":
		m.layout = LayoutUnified
		m.rememberLayout()
	case "2":
		m.layout = LayoutSplit
		m.rememberLayout()
	case "r":
		m.reloadAndReset()
	case "c":
		if m.cmd.Kind == KindHistory && len(m.view.Commits) > 0 {
			m.lastHistory = &m.cmd
			m.cmd = Command{Kind: KindCommit, Commit: m.view.Commits[m.cursor].Hash, Options: Options{Layout: m.layout}}
			m.reloadAndReset()
		}
	case "a", "p", "d":
		if m.cmd.Kind == KindStash {
			m.prepareStashAction(key)
		}
	}
	return m, nil
}

func (m tuiModel) updateOverlayKey(key string) (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlayModes, overlayBranch, overlayFile, overlayChangedFiles:
		switch key {
		case "esc":
			m.closeOverlay()
		case "backspace":
			m.removePickerFilterRune()
		case "up":
			if m.pickerCursor > 0 {
				m.pickerCursor--
			}
		case "down":
			if m.pickerCursor < len(m.pickerItems)-1 {
				m.pickerCursor++
			}
		case "enter":
			m.choosePicker()
		default:
			if len([]rune(key)) == 1 {
				m.pickerFilter += key
				m.applyPickerFilter()
			}
		}
	case overlayPrompt:
		switch key {
		case "esc":
			m.closeOverlay()
		case "enter":
			m.submitPrompt()
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 {
				m.input += key
			}
		}
	case overlayConfirm:
		switch key {
		case "esc", "n":
			m.closeOverlay()
		case "enter", "y":
			out, err := m.git.Run(m.confirmCmd...)
			if err != nil {
				m.err = err.Error()
			} else {
				m.message = strings.TrimSpace(out)
				if m.message == "" {
					m.message = "Command completed."
				}
			}
			m.closeOverlay()
			m.reloadAndReset()
		}
	}
	return m, nil
}

func (m tuiModel) updateMouse(msg tea.MouseMsg) tuiModel {
	if m.overlay != overlayNone {
		return m.updateOverlayMouse(msg)
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.scroll > 0 {
			m.scroll--
		}
	case tea.MouseButtonWheelDown:
		m.scroll++
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			sidebarWidth := m.sidebarWidth()
			if msg.X < sidebarWidth && msg.Y > 1 && msg.Y < m.height-1 {
				idx := msg.Y - 2
				if idx >= 0 && idx < m.itemCount() {
					m.cursor = idx
					m.diffFocused = false
					m.hScroll = 0
					m.resetDiffExpansion()
					m.selectCurrentItem()
				}
			}
		}
	}
	return m
}

func (m tuiModel) updateOverlayMouse(msg tea.MouseMsg) tuiModel {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case tea.MouseButtonWheelDown:
		if m.pickerCursor < len(m.pickerItems)-1 {
			m.pickerCursor++
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m
		}
		switch m.overlay {
		case overlayModes, overlayBranch, overlayFile, overlayChangedFiles:
			_, top, _, _ := m.overlayBounds()
			idx := msg.Y - top - 5
			if idx >= 0 && idx < len(m.pickerItems) {
				m.pickerCursor = idx
				m.choosePicker()
			}
		case overlayConfirm:
			_, _, _, bottom := m.overlayBounds()
			if msg.Y >= bottom-2 {
				out, err := m.git.Run(m.confirmCmd...)
				if err != nil {
					m.err = err.Error()
				} else {
					m.message = strings.TrimSpace(out)
					if m.message == "" {
						m.message = "Command completed."
					}
				}
				m.closeOverlay()
				m.reloadAndReset()
			}
		}
	}
	return m
}

func (m tuiModel) View() string {
	if m.err != "" {
		return styleError.Render(m.err) + "\n\n" + styleBottom.Render("q quit")
	}
	top := m.renderTop()
	body := m.renderBody()
	bottom := m.renderBottom()
	view := lipgloss.JoinVertical(lipgloss.Left, top, body, bottom)
	if m.overlay != overlayNone {
		return placeOverlay(view, m.renderOverlay(), m.width, m.height)
	}
	return view
}

func (m tuiModel) renderTop() string {
	add, del := totalCounts(m.view.Files)
	title := m.view.Title
	if title == "" {
		title = string(m.cmd.Kind)
	}
	meta := fmt.Sprintf("%s  %s  +%d -%d", title, m.view.Subtitle, add, del)
	if m.view.RestrictedPath != "" {
		meta += "  file: " + m.view.RestrictedPath
	}
	return styleTop.Width(m.width).Render(truncate(meta, m.width-2))
}

func (m tuiModel) renderBody() string {
	bodyHeight := m.height - 2
	if bodyHeight < 5 {
		bodyHeight = 5
	}
	if !m.showSidebar {
		return m.renderMain(m.width, bodyHeight)
	}
	sidebarWidth := m.sidebarWidth()
	mainWidth := m.width - sidebarWidth
	if mainWidth < 30 {
		mainWidth = 30
	}
	main := m.renderMain(mainWidth, bodyHeight)
	sidebar := m.renderSidebar(sidebarWidth, bodyHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m tuiModel) renderSidebar(width, height int) string {
	var lines []string
	switch m.cmd.Kind {
	case KindHistory:
		lines = append(lines, styleMuted.Render("Commits"))
		for i, item := range m.view.Commits {
			line := truncate(item.Raw, width-3)
			if i == m.cursor {
				line = styleSelected.Width(width - 3).Render(line)
			}
			lines = append(lines, line)
		}
	case KindStash:
		lines = append(lines, styleMuted.Render("Stashes"))
		for i, item := range m.view.Stashes {
			line := truncate(item.Raw, width-3)
			if i == m.cursor {
				line = styleSelected.Width(width - 3).Render(line)
			}
			lines = append(lines, line)
		}
		lines = append(lines, "", styleMuted.Render("Files"))
		for _, file := range m.view.Files {
			lines = append(lines, truncate(renderFileLine(file, width-3), width-3))
		}
	default:
		lines = append(lines, styleMuted.Render("Files"))
		for i, file := range m.view.Files {
			line := truncate(renderFileLine(file, width-3), width-3)
			if i == m.cursor {
				line = styleSelected.Width(width - 3).Render(line)
			}
			lines = append(lines, line)
		}
		if len(m.view.Commits) > 0 {
			lines = append(lines, "", styleMuted.Render("Commits"))
			for _, commit := range m.view.Commits {
				lines = append(lines, truncate(commit.Raw, width-3))
			}
		}
	}
	if m.showGitCmd && strings.TrimSpace(m.view.GitCommand) != "" {
		commandBlock := []string{"", styleMuted.Render("Git"), truncate(m.view.GitCommand, width-3)}
		contentHeight := height - len(commandBlock)
		if contentHeight < 0 {
			contentHeight = 0
		}
		lines = append(limitLines(lines, contentHeight), commandBlock...)
	}
	return styleSidebar.Width(width - 2).Height(height).Render(strings.Join(limitLines(lines, height), "\n"))
}

func (m tuiModel) renderMain(width, height int) string {
	diffWidth := width - 2
	lines := RenderDiffViewportWithOptions(m.view.Diff, m.layout, diffWidth, m.hScroll, m.diffRenderOptions())
	if m.scroll > len(lines) {
		m.scroll = len(lines)
	}
	end := m.scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	visible := lines[m.scroll:end]
	if len(visible) == 0 && m.view.Message != "" {
		visible = []string{m.view.Message}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Padding(0, 1).Render(strings.Join(limitLines(visible, height), "\n"))
}

func (m tuiModel) renderBottom() string {
	var hints []string
	gitHint := "g git"
	if m.showGitCmd {
		gitHint = "g hide git"
	}
	if m.diffFocused {
		hints = []string{"q quit", "/ modes", "esc files", "j/k scroll", "h/l scroll", "z fold", "Z all", "enter files", "s sidebar", gitHint, "1 unified", "2 split", "r refresh"}
	} else {
		hints = []string{"q quit", "/ modes"}
		if m.canClearFileRestriction() {
			hints = append(hints, "esc clear file")
		}
		hints = append(hints, "j/k move", "enter diff", "s sidebar", gitHint, "1 unified", "2 split")
		if supportsFilePicker(m.cmd.Kind) && !m.canClearFileRestriction() {
			hints = append(hints, "f file")
		}
		hints = append(hints, "r refresh")
	}
	if !m.diffFocused && m.view.SourceIsCurrent && m.cmd.Kind == KindCompare {
		hints = append(hints, "l local")
	}
	if m.cmd.Kind == KindStash {
		hints = append(hints, "a apply", "p pop", "d drop")
	}
	if m.cmd.Kind == KindHistory {
		hints = append(hints, "c commit")
	}
	if m.message != "" {
		hints = append([]string{m.message}, hints...)
	}
	return styleBottom.Width(m.width).Render(truncate(strings.Join(hints, "  "), m.width-2))
}

func (m tuiModel) renderOverlay() string {
	switch m.overlay {
	case overlayModes, overlayBranch, overlayFile, overlayChangedFiles:
		var lines []string
		lines = append(lines, m.pickerTitle, "filter: "+m.pickerFilter, "")
		if len(m.pickerItems) == 0 {
			lines = append(lines, styleMuted.Render("  no matches"))
		} else {
			for i, item := range m.pickerItems {
				line := item
				if i == m.pickerCursor {
					line = styleSelected.Render("> " + item)
				} else {
					line = "  " + item
				}
				lines = append(lines, line)
			}
		}
		lines = append(lines, "", "type filter   up/down move   backspace edit   enter select   esc cancel")
		return stylePopup.Render(strings.Join(lines, "\n"))
	case overlayPrompt:
		return stylePopup.Render(m.promptTitle + "\n\n" + m.input + "\n\nenter submit   esc cancel")
	case overlayConfirm:
		return stylePopup.Render(m.confirmText + "\n\ngit " + strings.Join(m.confirmCmd, " ") + "\n\nenter/y confirm   esc/n cancel")
	default:
		return ""
	}
}

func (m *tuiModel) reload() error {
	view, err := m.controller.Build(m.cmd)
	if err != nil {
		return err
	}
	m.view = view
	if m.cursor >= m.itemCount() {
		m.cursor = max(0, m.itemCount()-1)
	}
	return nil
}

func (m *tuiModel) reloadAndReset() {
	m.scroll = 0
	m.hScroll = 0
	m.diffFocused = false
	m.resetDiffExpansion()
	m.cursor = 0
	if err := m.reload(); err != nil {
		var pickerErr pickerNeededError
		if strings.Contains(err.Error(), "selector required") || pickerErr.Kind != "" {
			m.err = err.Error()
			return
		}
		m.err = err.Error()
	}
}

func (m *tuiModel) rememberSidebar() {
	m.state.Sidebar = m.showSidebar
	m.state.SidebarSet = true
	m.saveState()
}

func (m *tuiModel) rememberLayout() {
	m.state.Layout = m.layout
	m.state.LayoutSet = true
	m.saveState()
}

func (m *tuiModel) saveState() {
	if err := SaveState(m.state); err != nil {
		m.message = "Could not save state: " + err.Error()
	}
}

func (m *tuiModel) moveCursor(delta int) {
	count := m.itemCount()
	if count == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= count {
		m.cursor = count - 1
	}
	m.scroll = 0
	m.hScroll = 0
	m.resetDiffExpansion()
	if m.cmd.Kind == KindHistory {
		m.loadHistoryCommit()
	} else if m.cmd.Kind == KindCommit {
		m.loadCommitFile()
	} else if m.cursor < len(m.view.Files) {
		m.loadSelectedFileDiff()
	}
	if m.cmd.Kind == KindStash {
		m.loadStash()
	}
}

func (m *tuiModel) focusDiff() {
	if strings.TrimSpace(m.view.Diff) == "" && m.view.Message == "" {
		return
	}
	m.diffFocused = true
}

func (m tuiModel) diffRenderOptions() DiffRenderOptions {
	return DiffRenderOptions{
		Expanded:  m.expandedDiff,
		ExpandAll: m.expandAllDiff,
	}
}

func (m *tuiModel) resetDiffExpansion() {
	m.expandedDiff = nil
	m.expandAllDiff = false
}

func (m *tuiModel) toggleNearestCollapsedContext() {
	if strings.TrimSpace(m.view.Diff) == "" {
		return
	}
	target, ok := nearestCollapsedTarget(CollapsedContextTargets(m.view.Diff, m.layout, DiffRenderOptions{}), m.scroll)
	if !ok {
		m.message = "No collapsible context."
		return
	}
	if m.expandedDiff == nil {
		m.expandedDiff = map[string]bool{}
	}
	if m.expandAllDiff {
		m.expandAllDiff = false
	}
	if m.expandedDiff[target.Key] {
		delete(m.expandedDiff, target.Key)
		m.message = "Collapsed context."
		return
	}
	m.expandedDiff[target.Key] = true
	m.message = "Expanded context."
}

func (m *tuiModel) toggleAllCollapsedContext() {
	if strings.TrimSpace(m.view.Diff) == "" {
		return
	}
	if len(CollapsedContextTargets(m.view.Diff, m.layout, DiffRenderOptions{})) == 0 {
		m.message = "No collapsible context."
		return
	}
	if m.expandAllDiff {
		m.expandAllDiff = false
		m.expandedDiff = nil
		m.message = "Collapsed all context."
		return
	}
	m.expandAllDiff = true
	m.expandedDiff = nil
	m.message = "Expanded all context."
}

func nearestCollapsedTarget(targets []CollapsedContextTarget, line int) (CollapsedContextTarget, bool) {
	if len(targets) == 0 {
		return CollapsedContextTarget{}, false
	}
	best := targets[0]
	bestDistance := absInt(best.Line - line)
	for _, target := range targets[1:] {
		distance := absInt(target.Line - line)
		if distance < bestDistance {
			best = target
			bestDistance = distance
		}
	}
	return best, true
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (m *tuiModel) selectCurrentItem() {
	switch m.cmd.Kind {
	case KindHistory:
		m.loadHistoryCommit()
	case KindStash:
		m.loadStash()
	case KindCommit:
		m.loadCommitFile()
	default:
		if m.cursor < len(m.view.Files) {
			m.loadSelectedFileDiff()
		}
	}
}

func (m *tuiModel) restrictToFile() {
	if m.view.RestrictedPath != "" {
		return
	}
	m.openFilePicker("Restrict to file", filePaths(m.view.Files))
}

func (m *tuiModel) clearFileRestriction() {
	if !m.canClearFileRestriction() {
		return
	}
	m.cmd.Options.FilePath = ""
	m.cmd.Options.FileFlag = false
	m.reloadAndReset()
}

func (m tuiModel) canClearFileRestriction() bool {
	return supportsFilePicker(m.cmd.Kind) && m.view.RestrictedPath != ""
}

func (m *tuiModel) openBranchPicker(title string) {
	items, err := m.git.Branches()
	if err != nil {
		m.err = err.Error()
		return
	}
	m.overlay = overlayBranch
	m.pickerTitle = title
	m.setPickerItems(items)
}

func (m *tuiModel) openFilePicker(title string, items []string) {
	m.overlay = overlayFile
	m.pickerTitle = title
	m.setPickerItems(items)
}

func (m *tuiModel) setPickerItems(items []string) {
	m.pickerAllItems = append([]string(nil), items...)
	m.pickerFilter = ""
	m.pickerCursor = 0
	m.applyPickerFilter()
}

func (m *tuiModel) applyPickerFilter() {
	query := strings.ToLower(strings.TrimSpace(m.pickerFilter))
	if query == "" {
		m.pickerItems = append([]string(nil), m.pickerAllItems...)
	} else {
		m.pickerItems = m.pickerItems[:0]
		for _, item := range m.pickerAllItems {
			if strings.Contains(strings.ToLower(item), query) {
				m.pickerItems = append(m.pickerItems, item)
			}
		}
	}
	if m.pickerCursor >= len(m.pickerItems) {
		m.pickerCursor = 0
	}
	if m.pickerCursor < 0 {
		m.pickerCursor = 0
	}
}

func (m *tuiModel) removePickerFilterRune() {
	if m.pickerFilter == "" {
		return
	}
	runes := []rune(m.pickerFilter)
	m.pickerFilter = string(runes[:len(runes)-1])
	m.applyPickerFilter()
}

func (m *tuiModel) choosePicker() {
	if m.pickerCursor < 0 || m.pickerCursor >= len(m.pickerItems) {
		return
	}
	value := m.pickerItems[m.pickerCursor]
	switch m.overlay {
	case overlayModes:
		m.chooseMode(value)
	case overlayBranch:
		m.cmd.Target = value
		m.closeOverlay()
		if m.cmd.Options.FileFlag && m.cmd.Options.FilePath == "" {
			m.reload()
			m.openFilePicker("Restrict to file", filePaths(m.view.Files))
			return
		}
		m.reloadAndReset()
	case overlayFile:
		if m.cmd.Kind == KindHistory {
			m.cmd.Path = value
		} else {
			m.cmd.Options.FileFlag = true
			m.cmd.Options.FilePath = value
		}
		m.closeOverlay()
		m.reloadAndReset()
	case overlayChangedFiles:
		anchor := ""
		if m.cursor < len(m.view.Commits) {
			anchor = m.view.Commits[m.cursor].Hash
		}
		m.cmd = Command{Kind: KindHistory, Path: value, Options: Options{Layout: m.layout}}
		m.closeOverlay()
		m.reloadAndReset()
		if anchor != "" {
			m.anchorCommit(anchor)
		}
	}
}

func (m *tuiModel) chooseMode(value string) {
	m.closeOverlay()
	switch value {
	case "Local changes":
		m.cmd = Command{Kind: KindLocal, Options: Options{Layout: m.layout}}
		m.reloadAndReset()
	case "Compare/review branches":
		m.cmd = Command{Kind: KindCompare, Options: Options{Layout: m.layout}, Interactive: true}
		m.openBranchPicker("Choose target branch")
	case "Ahead of upstream":
		m.cmd = Command{Kind: KindAhead, Options: Options{Layout: m.layout}}
		m.reloadAndReset()
	case "Behind upstream":
		m.cmd = Command{Kind: KindBehind, Options: Options{Layout: m.layout}}
		m.reloadAndReset()
	case "Recent commits":
		m.openPrompt(promptRecentCount, "Recent commit count")
	case "File compare":
		m.openPrompt(promptFileCompare, "File compare: <path> <ref> [to-ref]")
	case "Stash":
		m.cmd = Command{Kind: KindStash, Options: Options{Layout: m.layout}}
		m.reloadAndReset()
	case "File history":
		files, err := m.git.TrackedFiles()
		if err != nil {
			m.err = err.Error()
			return
		}
		m.cmd = Command{Kind: KindHistory, Options: Options{Layout: m.layout}}
		m.openFilePicker("Choose file for history", files)
	case "Commit view":
		m.openPrompt(promptCommit, "Commit ref")
	}
}

func (m *tuiModel) openPrompt(kind promptKind, title string) {
	m.overlay = overlayPrompt
	m.promptKind = kind
	m.promptTitle = title
	m.input = ""
}

func (m *tuiModel) submitPrompt() {
	input := strings.TrimSpace(m.input)
	m.closeOverlay()
	switch m.promptKind {
	case promptRecentCount:
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 {
			m.err = "recent count must be a positive integer"
			return
		}
		m.cmd = Command{Kind: KindRecent, Count: n, Options: Options{Layout: m.layout}}
	case promptFileCompare:
		parts := strings.Fields(input)
		if len(parts) != 2 && len(parts) != 3 {
			m.err = "file compare needs: <path> <ref> [to-ref]"
			return
		}
		m.cmd = Command{Kind: KindFile, Path: parts[0], Options: Options{Layout: m.layout}}
		if len(parts) == 2 {
			m.cmd.Ref = parts[1]
		} else {
			m.cmd.FromRef = parts[1]
			m.cmd.ToRef = parts[2]
		}
	case promptHistoryFile:
		m.cmd = Command{Kind: KindHistory, Path: input, Options: Options{Layout: m.layout}}
	case promptCommit:
		m.cmd = Command{Kind: KindCommit, Commit: input, Options: Options{Layout: m.layout}}
	}
	m.reloadAndReset()
}

func (m *tuiModel) prepareStashAction(key string) {
	if len(m.view.Stashes) == 0 || m.cursor >= len(m.view.Stashes) {
		return
	}
	ref := m.view.Stashes[m.cursor].Ref
	localDirty := ""
	if lines, err := m.git.StatusShort(); err == nil && len(lines) > 0 {
		localDirty = "\n\nLocal uncommitted changes exist; conflicts are possible."
	}
	switch key {
	case "a":
		m.confirmText = "Apply " + ref + "?" + localDirty
		m.confirmCmd = []string{"stash", "apply", ref}
	case "p":
		m.confirmText = "Pop " + ref + "?\nThis applies the stash and removes it if successful." + localDirty
		m.confirmCmd = []string{"stash", "pop", ref}
	case "d":
		m.confirmText = "Drop " + ref + "?\nThis permanently removes the stash."
		m.confirmCmd = []string{"stash", "drop", ref}
	}
	m.overlay = overlayConfirm
}

func (m *tuiModel) openChangedFilesForSelectedCommit() {
	if m.cursor >= len(m.view.Commits) {
		return
	}
	commit := m.view.Commits[m.cursor].Hash
	files, err := m.git.CommitChangedFiles(commit)
	if err != nil {
		m.err = err.Error()
		return
	}
	m.overlay = overlayChangedFiles
	m.pickerTitle = "Files changed in " + commit
	m.setPickerItems(files)
}

func (m *tuiModel) anchorCommit(hash string) {
	for i, commit := range m.view.Commits {
		if commit.Hash == hash {
			m.cursor = i
			m.loadHistoryCommit()
			return
		}
	}
}

func (m *tuiModel) loadHistoryCommit() {
	if m.cursor >= len(m.view.Commits) || m.cmd.Path == "" {
		return
	}
	commit := m.view.Commits[m.cursor].Hash
	out, err := m.git.ShowForTUI(commit, "--", m.cmd.Path)
	if err != nil {
		m.err = err.Error()
		return
	}
	m.view.Diff = out
	m.view.GitCommand = showCommandForView(m.cmd, commit, "--", m.cmd.Path)
}

func (m *tuiModel) loadCommitFile() {
	if m.cursor >= len(m.view.Files) {
		return
	}
	file := m.view.Files[m.cursor].Path
	out, err := m.git.ShowForTUI(m.cmd.Commit, "--", file)
	if err != nil {
		m.err = err.Error()
		return
	}
	m.view.Diff = out
	m.view.GitCommand = showCommandForView(m.cmd, m.cmd.Commit, "--", file)
}

func (m *tuiModel) loadSelectedFileDiff() {
	if m.cursor >= len(m.view.Files) {
		return
	}
	path := m.view.Files[m.cursor].Path
	out, command, err := m.diffForPath(path)
	if err != nil {
		m.err = err.Error()
		return
	}
	m.view.Diff = out
	m.view.GitCommand = command
}

func (m *tuiModel) diffForPath(path string) (string, string, error) {
	switch m.cmd.Kind {
	case KindLocal:
		if m.cmd.Options.LocalMode == "staged" {
			args := []string{"--staged", "--", path}
			out, err := m.git.DiffForTUI(args...)
			return out, diffCommandForView(m.cmd, args...), err
		}
		args := []string{"--", path}
		out, err := m.git.DiffForTUI(args...)
		if strings.TrimSpace(out) == "" {
			if untracked, untrackedErr := m.git.UntrackedPatchForTUI(path); untrackedErr == nil {
				return untracked, "git diff --no-ext-diff --no-index --unified=" + diffyTUIContextLines + " -- /dev/null " + path, nil
			}
		}
		return out, diffCommandForView(m.cmd, args...), err
	case KindCompare:
		if m.cmd.Target == "" {
			return m.view.Diff, m.view.GitCommand, nil
		}
		target, err := m.git.ResolveRef(m.cmd.Target)
		if err != nil {
			return "", "", err
		}
		source := "HEAD"
		if m.cmd.Source != "" {
			source, err = m.git.ResolveRef(m.cmd.Source)
			if err != nil {
				return "", "", err
			}
		}
		arg := target + "..." + source
		args := []string{arg, "--", path}
		out, err := m.git.DiffForTUI(args...)
		return out, diffCommandForView(m.cmd, args...), err
	case KindAhead:
		upstream, err := m.git.Upstream()
		if err != nil {
			return "", "", err
		}
		arg := upstream + "...HEAD"
		args := []string{arg, "--", path}
		out, err := m.git.DiffForTUI(args...)
		return out, diffCommandForView(m.cmd, args...), err
	case KindBehind:
		upstream, err := m.git.Upstream()
		if err != nil {
			return "", "", err
		}
		arg := "HEAD.." + upstream
		args := []string{arg, "--", path}
		out, err := m.git.DiffForTUI(args...)
		return out, diffCommandForView(m.cmd, args...), err
	case KindRecent:
		arg := fmt.Sprintf("HEAD~%d..HEAD", m.cmd.Count)
		args := []string{arg, "--", path}
		out, err := m.git.DiffForTUI(args...)
		return out, diffCommandForView(m.cmd, args...), err
	case KindFile:
		return m.view.Diff, m.view.GitCommand, nil
	default:
		return m.view.Diff, m.view.GitCommand, nil
	}
}

func (m *tuiModel) loadStash() {
	if m.cursor >= len(m.view.Stashes) {
		return
	}
	ref := m.view.Stashes[m.cursor].Ref
	m.cmd.StashRef = ref
	m.reloadAndReset()
}

func (m *tuiModel) closeOverlay() {
	m.overlay = overlayNone
	m.pickerItems = nil
	m.pickerAllItems = nil
	m.pickerFilter = ""
	m.input = ""
}

func (m tuiModel) itemCount() int {
	switch m.cmd.Kind {
	case KindHistory:
		return len(m.view.Commits)
	case KindStash:
		return len(m.view.Stashes)
	default:
		return len(m.view.Files)
	}
}

func (m tuiModel) sidebarWidth() int {
	if !m.showSidebar {
		return 0
	}
	width := m.width / 3
	if width < 28 {
		return 28
	}
	if width > 48 {
		return 48
	}
	return width
}

func filePaths(files []FileStat) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func supportsFilePicker(kind CommandKind) bool {
	switch kind {
	case KindLocal, KindCompare, KindRecent:
		return true
	default:
		return false
	}
}

func limitLines(lines []string, n int) []string {
	if len(lines) > n {
		return lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}

func placeOverlay(base, overlay string, width, height int) string {
	if width < 1 || height < 1 {
		return overlay
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("236")))
}

func (m tuiModel) overlayBounds() (left, top, right, bottom int) {
	overlay := m.renderOverlay()
	overlayWidth := lipgloss.Width(overlay)
	overlayHeight := lipgloss.Height(overlay)
	left = (m.width - overlayWidth) / 2
	top = (m.height - overlayHeight) / 2
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	return left, top, left + overlayWidth, top + overlayHeight
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
