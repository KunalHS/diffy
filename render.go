package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

var (
	styleTop       = lipgloss.NewStyle().Background(lipgloss.Color("#2B3442")).Foreground(lipgloss.Color("#F8FAFC")).Padding(0, 1)
	styleBottom    = lipgloss.NewStyle().Background(lipgloss.Color("#202A37")).Foreground(lipgloss.Color("#CBD5E1")).Padding(0, 1)
	styleSidebar   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(lipgloss.Color("#475569")).Padding(0, 1)
	styleSelected  = lipgloss.NewStyle().Background(lipgloss.Color("#E2E8F0")).Foreground(lipgloss.Color("#0F172A"))
	styleMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	styleAdd       = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	styleDel       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FB7185"))
	styleAddLine   = lipgloss.NewStyle().Foreground(lipgloss.Color("#DCFCE7")).Background(lipgloss.Color("#14532D"))
	styleDelLine   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE4E6")).Background(lipgloss.Color("#5F1E2E"))
	styleHunk      = lipgloss.NewStyle().Foreground(lipgloss.Color("#93C5FD"))
	styleHunkLine  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BFDBFE")).Background(lipgloss.Color("#1E293B"))
	stylePopup     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#A78BFA")).Padding(1, 2)
	styleWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FB7185"))
	stylePaneTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0F2FE")).Background(lipgloss.Color("#1E3A5F")).Bold(true)
	styleGutter    = lipgloss.NewStyle().Foreground(lipgloss.Color("#93A4B8"))
	styleDivider   = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	styleCollapsed = lipgloss.NewStyle().Foreground(lipgloss.Color("#FDE68A")).Background(lipgloss.Color("#334155")).Bold(true)
)

func RenderDiff(diff string, layout Layout, width int) []string {
	return RenderDiffViewport(diff, layout, width, 0)
}

func RenderDiffViewport(diff string, layout Layout, width, hScroll int) []string {
	return RenderDiffViewportWithOptions(diff, layout, width, hScroll, DiffRenderOptions{})
}

func RenderDiffViewportWithOptions(diff string, layout Layout, width, hScroll int, opts DiffRenderOptions) []string {
	if width < 20 {
		width = 20
	}
	if strings.TrimSpace(diff) == "" {
		return []string{styleMuted.Render("No diff to show.")}
	}
	if hScroll < 0 {
		hScroll = 0
	}

	doc := ParseDiffDocument(diff)
	if len(doc.Files) == 0 {
		return []string{styleMuted.Render("No textual hunks to show.")}
	}
	if layout == LayoutSplit {
		return renderSplitDocument(doc, width, hScroll, opts)
	}
	return renderUnifiedDocument(doc, width, hScroll, opts)
}

func renderSplitDocument(doc DiffDocument, width, hScroll int, opts DiffRenderOptions) []string {
	gutterWidth := gutterWidthForDocument(doc)
	dividerWidth := 3
	paneWidth := (width - dividerWidth) / 2
	if paneWidth < gutterWidth+12 {
		return renderUnifiedDocument(doc, width, hScroll, opts)
	}
	rightPaneWidth := width - paneWidth - dividerWidth

	var lines []string
	for fileIndex, file := range doc.Files {
		if fileIndex > 0 && len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, renderSplitFileHeader(file, paneWidth, rightPaneWidth))
		rows := VisibleRowsForFile(file, fileIndex, opts)
		if len(rows) == 0 {
			lines = append(lines, styleMuted.Width(width).Render("No textual hunks in this file."))
			continue
		}
		for _, row := range rows {
			lines = append(lines, renderSplitRow(row, gutterWidth, paneWidth, rightPaneWidth, width, hScroll))
		}
	}
	if len(lines) == 0 {
		return []string{styleMuted.Render("No textual hunks to show.")}
	}
	return lines
}

func renderSplitFileHeader(file DiffFile, leftWidth, rightWidth int) string {
	oldName, newName := displayFileNames(file)
	left := stylePaneTitle.Width(leftWidth).Render(padDisplay(cutDisplay("old  "+oldName, 0, leftWidth), leftWidth))
	right := stylePaneTitle.Width(rightWidth).Render(padDisplay(cutDisplay("new  "+newName, 0, rightWidth), rightWidth))
	return left + styleDivider.Render(" │ ") + right
}

func renderSplitRow(row DiffRow, gutterWidth, leftWidth, rightWidth, totalWidth, hScroll int) string {
	if row.Kind == RowCollapsed && row.Collapsed != nil {
		return styleCollapsed.Width(totalWidth).Render(padDisplay(collapsedLabel(row.Collapsed), totalWidth))
	}
	if row.Kind == RowNote {
		return styleMuted.Width(totalWidth).Render(padDisplay(row.Note, totalWidth))
	}

	leftKind := RowContext
	rightKind := RowContext
	leftLine := row.OldLine
	rightLine := row.NewLine
	leftText := row.OldText
	rightText := row.NewText
	leftBlank := false
	rightBlank := false

	switch row.Kind {
	case RowAdd:
		leftBlank = true
		rightKind = RowAdd
	case RowDelete:
		leftKind = RowDelete
		rightBlank = true
	case RowModify:
		leftKind = RowDelete
		rightKind = RowAdd
	case RowContext:
		// Defaults are already set.
	default:
		leftBlank = true
		rightBlank = true
	}

	left := renderSplitCell(leftKind, leftLine, leftText, leftBlank, gutterWidth, leftWidth, hScroll)
	right := renderSplitCell(rightKind, rightLine, rightText, rightBlank, gutterWidth, rightWidth, hScroll)
	return left + styleDivider.Render(" │ ") + right
}

func renderSplitCell(kind DiffRowKind, lineNumber int, text string, blank bool, gutterWidth, paneWidth, hScroll int) string {
	codeWidth := paneWidth - gutterWidth - 3
	if codeWidth < 1 {
		codeWidth = 1
	}
	marker := " "
	switch kind {
	case RowAdd:
		marker = "+"
	case RowDelete:
		marker = "-"
	}
	number := formatLineNumber(lineNumber, gutterWidth)
	if blank {
		marker = " "
		number = strings.Repeat(" ", gutterWidth)
		text = ""
	}
	gutter := styleGutter.Render(marker + " " + number + " ")
	code := padDisplay(cutDisplay(text, hScroll, codeWidth), codeWidth)
	cell := gutter + code
	switch kind {
	case RowAdd:
		return styleAddLine.Width(paneWidth).Render(cell)
	case RowDelete:
		return styleDelLine.Width(paneWidth).Render(cell)
	default:
		return padDisplay(cell, paneWidth)
	}
}

func renderUnifiedDocument(doc DiffDocument, width, hScroll int, opts DiffRenderOptions) []string {
	gutterWidth := gutterWidthForDocument(doc)
	var lines []string
	for fileIndex, file := range doc.Files {
		if fileIndex > 0 && len(lines) > 0 {
			lines = append(lines, "")
		}
		_, newName := displayFileNames(file)
		lines = append(lines, stylePaneTitle.Width(width).Render(padDisplay(cutDisplay(newName, 0, width), width)))
		rows := VisibleRowsForFile(file, fileIndex, opts)
		if len(rows) == 0 {
			lines = append(lines, styleMuted.Width(width).Render("No textual hunks in this file."))
			continue
		}
		for _, row := range rows {
			lines = append(lines, renderUnifiedRow(row, gutterWidth, width, hScroll)...)
		}
	}
	if len(lines) == 0 {
		return []string{styleMuted.Render("No textual hunks to show.")}
	}
	return lines
}

func renderUnifiedRow(row DiffRow, gutterWidth, width, hScroll int) []string {
	if row.Kind == RowCollapsed && row.Collapsed != nil {
		return []string{styleCollapsed.Width(width).Render(padDisplay(collapsedLabel(row.Collapsed), width))}
	}
	if row.Kind == RowNote {
		return []string{styleMuted.Width(width).Render(padDisplay(row.Note, width))}
	}
	if row.Kind == RowModify {
		return []string{
			renderUnifiedCell(RowDelete, row.OldLine, 0, row.OldText, gutterWidth, width, hScroll),
			renderUnifiedCell(RowAdd, 0, row.NewLine, row.NewText, gutterWidth, width, hScroll),
		}
	}
	switch row.Kind {
	case RowAdd:
		return []string{renderUnifiedCell(RowAdd, 0, row.NewLine, row.NewText, gutterWidth, width, hScroll)}
	case RowDelete:
		return []string{renderUnifiedCell(RowDelete, row.OldLine, 0, row.OldText, gutterWidth, width, hScroll)}
	default:
		return []string{renderUnifiedCell(RowContext, row.OldLine, row.NewLine, row.NewText, gutterWidth, width, hScroll)}
	}
}

func renderUnifiedCell(kind DiffRowKind, oldLine, newLine int, text string, gutterWidth, width, hScroll int) string {
	marker := " "
	switch kind {
	case RowAdd:
		marker = "+"
	case RowDelete:
		marker = "-"
	}
	gutter := styleGutter.Render(formatLineNumber(oldLine, gutterWidth) + " " + formatLineNumber(newLine, gutterWidth) + " " + marker + " ")
	codeWidth := width - lipgloss.Width(gutter)
	if codeWidth < 1 {
		codeWidth = 1
	}
	code := padDisplay(cutDisplay(text, hScroll, codeWidth), codeWidth)
	line := gutter + code
	switch kind {
	case RowAdd:
		return styleAddLine.Width(width).Render(line)
	case RowDelete:
		return styleDelLine.Width(width).Render(line)
	default:
		return padDisplay(line, width)
	}
}

func displayFileNames(file DiffFile) (string, string) {
	oldName := file.OldPath
	newName := file.NewPath
	if oldName == "" {
		oldName = "/dev/null"
	}
	if newName == "" {
		newName = oldName
	}
	return oldName, newName
}

func collapsedLabel(context *CollapsedContext) string {
	if context.Count == 1 {
		return "... 1 unchanged line ..."
	}
	return fmt.Sprintf("... %d unchanged lines ...", context.Count)
}

func gutterWidthForDocument(doc DiffDocument) int {
	maxLine := 0
	for _, file := range doc.Files {
		for _, row := range file.Rows {
			if row.OldLine > maxLine {
				maxLine = row.OldLine
			}
			if row.NewLine > maxLine {
				maxLine = row.NewLine
			}
		}
	}
	width := len(strconv.Itoa(maxLine))
	if width < 3 {
		width = 3
	}
	return width
}

func formatLineNumber(line, width int) string {
	if line <= 0 {
		return strings.Repeat(" ", width)
	}
	value := strconv.Itoa(line)
	if lipgloss.Width(value) >= width {
		return value
	}
	return strings.Repeat(" ", width-lipgloss.Width(value)) + value
}

func renderFileLine(file FileStat, width int) string {
	name := truncate(file.Path, width-12)
	if file.Binary {
		return fmt.Sprintf("%s binary", name)
	}
	return fmt.Sprintf("%s +%d -%d", name, file.Add, file.Del)
}

func truncate(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 1 {
		return s[:width]
	}
	return s[:width-1] + "~"
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func cutDisplay(s string, offset, width int) string {
	if width <= 0 {
		return ""
	}
	if offset < 0 {
		offset = 0
	}
	return xansi.Cut(s, offset, offset+width)
}

func padDisplay(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if actual := lipgloss.Width(s); actual < width {
		return s + strings.Repeat(" ", width-actual)
	}
	return s
}
