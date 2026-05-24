package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

var (
	colorAddLineBG = lipgloss.Color("#14532D")
	colorDelLineBG = lipgloss.Color("#5F1E2E")

	styleTop       = lipgloss.NewStyle().Background(lipgloss.Color("#2B3442")).Foreground(lipgloss.Color("#F8FAFC")).Padding(0, 1)
	styleBottom    = lipgloss.NewStyle().Background(lipgloss.Color("#202A37")).Foreground(lipgloss.Color("#CBD5E1")).Padding(0, 1)
	styleSidebar   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(lipgloss.Color("#475569")).Padding(0, 1)
	styleSelected  = lipgloss.NewStyle().Background(lipgloss.Color("#E2E8F0")).Foreground(lipgloss.Color("#0F172A"))
	styleMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	styleAdd       = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	styleDel       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FB7185"))
	styleAddLine   = lipgloss.NewStyle().Foreground(lipgloss.Color("#DCFCE7")).Background(colorAddLineBG)
	styleDelLine   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE4E6")).Background(colorDelLineBG)
	styleHunk      = lipgloss.NewStyle().Foreground(lipgloss.Color("#93C5FD"))
	styleHunkLine  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BFDBFE")).Background(lipgloss.Color("#1E293B"))
	stylePopup     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#A78BFA")).Padding(1, 2)
	styleWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FB7185"))
	stylePaneTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0F2FE")).Background(lipgloss.Color("#1E3A5F")).Bold(true)
	styleGutter    = lipgloss.NewStyle().Foreground(lipgloss.Color("#93A4B8"))
	styleAddGutter = lipgloss.NewStyle().Foreground(lipgloss.Color("#A7F3D0")).Background(colorAddLineBG)
	styleDelGutter = lipgloss.NewStyle().Foreground(lipgloss.Color("#FDA4AF")).Background(colorDelLineBG)
	styleDivider   = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	styleCollapsed = lipgloss.NewStyle().Foreground(lipgloss.Color("#FDE68A")).Background(lipgloss.Color("#334155")).Bold(true)
	styleAddInline = lipgloss.NewStyle().Foreground(lipgloss.Color("#F0FDF4")).Background(lipgloss.Color("#15803D")).Bold(true)
	styleDelInline = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF1F2")).Background(lipgloss.Color("#BE123C")).Bold(true)
	styleLane      = lipgloss.NewStyle().Foreground(lipgloss.Color("#CBD5E1")).Background(lipgloss.Color("#273449")).Bold(true)
	styleLaneIdle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	styleFiller    = lipgloss.NewStyle().Background(lipgloss.Color("#1F2937"))
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
		rows := PrepareRenderRows(VisibleRowsForFile(file, fileIndex, opts))
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

func renderSplitRow(renderRow RenderRow, gutterWidth, leftWidth, rightWidth, totalWidth, hScroll int) string {
	row := renderRow.Row
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

	left := renderSplitCell(leftKind, leftLine, leftText, leftBlank, gutterWidth, leftWidth, hScroll, renderRow.OldInline)
	right := renderSplitCell(rightKind, rightLine, rightText, rightBlank, gutterWidth, rightWidth, hScroll, renderRow.NewInline)
	return left + renderChangeLane(renderRow.BlockPosition) + right
}

func renderSplitCell(kind DiffRowKind, lineNumber int, text string, blank bool, gutterWidth, paneWidth, hScroll int, spans []InlineSpan) string {
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
		spans = nil
	}
	gutterText := marker + " " + number + " "
	switch kind {
	case RowAdd:
		return renderChangedCell(RowAdd, gutterText, text, hScroll, codeWidth, spans)
	case RowDelete:
		return renderChangedCell(RowDelete, gutterText, text, hScroll, codeWidth, spans)
	default:
		gutter := styleGutter.Render(gutterText)
		code := renderCodeText(text, hScroll, codeWidth, spans, inlineStyleForKind(kind))
		cell := gutter + code
		if blank {
			return styleFiller.Width(paneWidth).Render(padDisplay(cell, paneWidth))
		}
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
		rows := PrepareRenderRows(VisibleRowsForFile(file, fileIndex, opts))
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

func renderUnifiedRow(renderRow RenderRow, gutterWidth, width, hScroll int) []string {
	row := renderRow.Row
	if row.Kind == RowCollapsed && row.Collapsed != nil {
		return []string{styleCollapsed.Width(width).Render(padDisplay(collapsedLabel(row.Collapsed), width))}
	}
	if row.Kind == RowNote {
		return []string{styleMuted.Width(width).Render(padDisplay(row.Note, width))}
	}
	if row.Kind == RowModify {
		return []string{
			renderUnifiedCell(RowDelete, row.OldLine, 0, row.OldText, gutterWidth, width, hScroll, renderRow.OldInline),
			renderUnifiedCell(RowAdd, 0, row.NewLine, row.NewText, gutterWidth, width, hScroll, renderRow.NewInline),
		}
	}
	switch row.Kind {
	case RowAdd:
		return []string{renderUnifiedCell(RowAdd, 0, row.NewLine, row.NewText, gutterWidth, width, hScroll, nil)}
	case RowDelete:
		return []string{renderUnifiedCell(RowDelete, row.OldLine, 0, row.OldText, gutterWidth, width, hScroll, nil)}
	default:
		return []string{renderUnifiedCell(RowContext, row.OldLine, row.NewLine, row.NewText, gutterWidth, width, hScroll, nil)}
	}
}

func renderUnifiedCell(kind DiffRowKind, oldLine, newLine int, text string, gutterWidth, width, hScroll int, spans []InlineSpan) string {
	marker := " "
	switch kind {
	case RowAdd:
		marker = "+"
	case RowDelete:
		marker = "-"
	}
	gutterText := formatLineNumber(oldLine, gutterWidth) + " " + formatLineNumber(newLine, gutterWidth) + " " + marker + " "
	codeWidth := width - lipgloss.Width(gutterText)
	if codeWidth < 1 {
		codeWidth = 1
	}
	switch kind {
	case RowAdd:
		return renderChangedCell(RowAdd, gutterText, text, hScroll, codeWidth, spans)
	case RowDelete:
		return renderChangedCell(RowDelete, gutterText, text, hScroll, codeWidth, spans)
	default:
		gutter := styleGutter.Render(gutterText)
		code := renderCodeText(text, hScroll, codeWidth, spans, inlineStyleForKind(kind))
		line := gutter + code
		return padDisplay(line, width)
	}
}

func renderChangedCell(kind DiffRowKind, gutterText, text string, hScroll, codeWidth int, spans []InlineSpan) string {
	lineStyle := lineStyleForKind(kind)
	gutterStyle := gutterStyleForKind(kind)
	code := renderStyledCodeText(text, hScroll, codeWidth, spans, lineStyle, inlineStyleForKind(kind))
	return gutterStyle.Render(gutterText) + code
}

func renderChangeLane(position BlockPosition) string {
	marker := "│"
	style := styleLaneIdle
	switch position {
	case BlockStart:
		marker = "╭"
		style = styleLane
	case BlockMiddle:
		marker = "┃"
		style = styleLane
	case BlockEnd:
		marker = "╰"
		style = styleLane
	case BlockSingle:
		marker = "◆"
		style = styleLane
	}
	return style.Width(3).Render(" " + marker + " ")
}

func inlineStyleForKind(kind DiffRowKind) lipgloss.Style {
	if kind == RowAdd {
		return styleAddInline
	}
	return styleDelInline
}

func lineStyleForKind(kind DiffRowKind) lipgloss.Style {
	if kind == RowAdd {
		return styleAddLine
	}
	return styleDelLine
}

func gutterStyleForKind(kind DiffRowKind) lipgloss.Style {
	if kind == RowAdd {
		return styleAddGutter
	}
	return styleDelGutter
}

func renderCodeText(text string, hScroll, width int, spans []InlineSpan, highlightStyle lipgloss.Style) string {
	return renderCodeTextWithBase(text, hScroll, width, spans, lipgloss.NewStyle(), highlightStyle)
}

func renderStyledCodeText(text string, hScroll, width int, spans []InlineSpan, baseStyle, highlightStyle lipgloss.Style) string {
	return renderCodeTextWithBase(text, hScroll, width, spans, baseStyle, highlightStyle)
}

func renderCodeTextWithBase(text string, hScroll, width int, spans []InlineSpan, baseStyle, highlightStyle lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if hScroll < 0 {
		hScroll = 0
	}
	if hScroll > len(runes) {
		hScroll = len(runes)
	}
	end := hScroll + width
	if end > len(runes) {
		end = len(runes)
	}
	var builder strings.Builder
	for index := hScroll; index < end; {
		span, ok := spanCovering(spans, index)
		next := end
		if ok {
			if span.End < next {
				next = span.End
			}
			builder.WriteString(highlightStyle.Render(string(runes[index:next])))
		} else {
			if boundary := nextSpanStart(spans, index, end); boundary < next {
				next = boundary
			}
			builder.WriteString(baseStyle.Render(string(runes[index:next])))
		}
		index = next
	}
	if actual := lipgloss.Width(builder.String()); actual < width {
		builder.WriteString(baseStyle.Render(strings.Repeat(" ", width-actual)))
	}
	return builder.String()
}

func spanCovering(spans []InlineSpan, index int) (InlineSpan, bool) {
	for _, span := range spans {
		if index >= span.Start && index < span.End {
			return span, true
		}
	}
	return InlineSpan{}, false
}

func nextSpanStart(spans []InlineSpan, index, end int) int {
	next := end
	for _, span := range spans {
		if span.Start > index && span.Start < next {
			next = span.Start
		}
	}
	return next
}

func displayFileNames(file DiffFile) (string, string) {
	oldName := displayDiffFileName(file.OldPath)
	newName := displayDiffFileName(file.NewPath)
	if oldName == "" {
		oldName = "/dev/null"
	}
	if newName == "" {
		newName = oldName
	}
	return oldName, newName
}

func displayDiffFileName(filePath string) string {
	if filePath == "" || filePath == "/dev/null" {
		return filePath
	}
	return sidebarFileName(filePath)
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
	name := truncate(sidebarFileName(file.Path), width-12)
	if file.Binary {
		return fmt.Sprintf("%s binary", name)
	}
	return fmt.Sprintf("%s +%d -%d", name, file.Add, file.Del)
}

func sidebarFileName(filePath string) string {
	name := path.Base(filePath)
	if name == "." || name == "/" {
		return filePath
	}
	return name
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
