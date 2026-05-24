package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

var (
	styleTop      = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252")).Padding(0, 1)
	styleBottom   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("245")).Padding(0, 1)
	styleSidebar  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("250")).Foreground(lipgloss.Color("235"))
	styleMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleAdd      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleDel      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleAddLine  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Background(lipgloss.Color("22"))
	styleDelLine  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Background(lipgloss.Color("52"))
	styleHunk     = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	styleHunkLine = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Background(lipgloss.Color("236"))
	stylePopup    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("99")).Padding(1, 2)
	styleWarn     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

type renderLineKind int

const (
	renderLineContext renderLineKind = iota
	renderLineAdd
	renderLineDelete
	renderLineNote
)

type renderFile struct {
	IsNew     bool
	IsDeleted bool
	Hunks     []renderHunk
}

type renderHunk struct {
	Header string
	Lines  []renderLine
}

type renderLine struct {
	Kind renderLineKind
	Text string
}

func RenderDiff(diff string, layout Layout, width int) []string {
	return RenderDiffViewport(diff, layout, width, 0)
}

func RenderDiffViewport(diff string, layout Layout, width, hScroll int) []string {
	if width < 20 {
		width = 20
	}
	if strings.TrimSpace(diff) == "" {
		return []string{styleMuted.Render("No diff to show.")}
	}
	if hScroll < 0 {
		hScroll = 0
	}
	if layout == LayoutSplit {
		return renderSplit(diff, width, hScroll)
	}
	return renderUnified(diff, width, hScroll)
}

func renderUnified(diff string, width, hScroll int) []string {
	var lines []string
	for _, file := range parseDiffForRender(diff) {
		for _, hunk := range file.Hunks {
			lines = append(lines, styleHunkLine.Width(width).Render(cutDisplay(hunk.Header, 0, width)))
			for _, line := range hunk.Lines {
				lines = append(lines, colorRenderLine(line, width, hScroll))
			}
		}
	}
	if len(lines) == 0 {
		return []string{styleMuted.Render("No textual hunks to show.")}
	}
	return lines
}

func renderSplit(diff string, width, hScroll int) []string {
	leftWidth := (width - 5) / 2
	if leftWidth < 20 {
		return renderUnified(diff, width, hScroll)
	}
	rightWidth := width - leftWidth - 5
	var lines []string
	for fileIndex, file := range parseDiffForRender(diff) {
		if fileIndex > 0 && len(file.Hunks) > 0 && len(lines) > 0 {
			lines = append(lines, "")
		}
		for _, hunk := range file.Hunks {
			lines = append(lines, styleHunkLine.Width(width).Render(cutDisplay(hunk.Header, 0, width)))
			lines = append(lines, renderSplitHunk(file, hunk, width, leftWidth, rightWidth, hScroll)...)
		}
	}
	if len(lines) == 0 {
		return []string{styleMuted.Render("No textual hunks to show.")}
	}
	return lines
}

func renderSplitHunk(file renderFile, hunk renderHunk, width, leftWidth, rightWidth, hScroll int) []string {
	var lines []string
	for i := 0; i < len(hunk.Lines); {
		line := hunk.Lines[i]
		switch line.Kind {
		case renderLineDelete:
			var deleted []renderLine
			for i < len(hunk.Lines) && hunk.Lines[i].Kind == renderLineDelete {
				deleted = append(deleted, hunk.Lines[i])
				i++
			}
			var added []renderLine
			for i < len(hunk.Lines) && hunk.Lines[i].Kind == renderLineAdd {
				added = append(added, hunk.Lines[i])
				i++
			}
			if file.IsDeleted {
				for _, deletedLine := range deleted {
					lines = append(lines, fullDiffLine(deletedLine, width, hScroll))
				}
				continue
			}
			if file.IsNew {
				for _, addedLine := range added {
					lines = append(lines, fullDiffLine(addedLine, width, hScroll))
				}
				continue
			}
			count := max(len(deleted), len(added))
			for j := 0; j < count; j++ {
				left := ""
				right := ""
				if j < len(deleted) {
					left = deleted[j].Text
				}
				if j < len(added) {
					right = added[j].Text
				}
				lines = append(lines, splitDiffLine(left, right, leftWidth, rightWidth, hScroll))
			}
		case renderLineAdd:
			var added []renderLine
			for i < len(hunk.Lines) && hunk.Lines[i].Kind == renderLineAdd {
				added = append(added, hunk.Lines[i])
				i++
			}
			for _, addedLine := range added {
				if file.IsNew {
					lines = append(lines, fullDiffLine(addedLine, width, hScroll))
				} else {
					lines = append(lines, splitDiffLine("", addedLine.Text, leftWidth, rightWidth, hScroll))
				}
			}
		case renderLineContext:
			value := strings.TrimPrefix(line.Text, " ")
			lines = append(lines, splitContextLine(value, leftWidth, rightWidth, hScroll))
			i++
		case renderLineNote:
			lines = append(lines, styleMuted.Render(cutDisplay(line.Text, hScroll, width)))
			i++
		}
	}
	return lines
}

func splitDiffLine(leftText, rightText string, leftWidth, rightWidth, hScroll int) string {
	left := padDisplay(cutDisplay(leftText, hScroll, leftWidth), leftWidth)
	right := padDisplay(cutDisplay(rightText, hScroll, rightWidth), rightWidth)
	if leftText != "" {
		left = styleDelLine.Width(leftWidth).Render(left)
	}
	if rightText != "" {
		right = styleAddLine.Width(rightWidth).Render(right)
	}
	return left + "  |  " + right
}

func splitContextLine(text string, leftWidth, rightWidth, hScroll int) string {
	left := padDisplay(cutDisplay(text, hScroll, leftWidth), leftWidth)
	right := padDisplay(cutDisplay(text, hScroll, rightWidth), rightWidth)
	return left + "  |  " + right
}

func fullDiffLine(line renderLine, width, hScroll int) string {
	text := padDisplay(cutDisplay(line.Text, hScroll, width), width)
	switch line.Kind {
	case renderLineAdd:
		return styleAddLine.Width(width).Render(text)
	case renderLineDelete:
		return styleDelLine.Width(width).Render(text)
	case renderLineNote:
		return styleMuted.Render(text)
	default:
		return text
	}
}

func parseDiffForRender(diff string) []renderFile {
	var files []renderFile
	currentFile := -1
	currentHunk := -1

	ensureFile := func() int {
		if currentFile < 0 {
			files = append(files, renderFile{})
			currentFile = len(files) - 1
		}
		return currentFile
	}

	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			files = append(files, renderFile{})
			currentFile = len(files) - 1
			currentHunk = -1
			continue
		}
		if strings.HasPrefix(line, "@@") {
			fileIndex := ensureFile()
			files[fileIndex].Hunks = append(files[fileIndex].Hunks, renderHunk{Header: line})
			currentHunk = len(files[fileIndex].Hunks) - 1
			continue
		}
		if currentFile >= 0 && currentHunk < 0 {
			switch line {
			case "--- /dev/null":
				files[currentFile].IsNew = true
			case "+++ /dev/null":
				files[currentFile].IsDeleted = true
			}
			continue
		}
		if currentFile < 0 || currentHunk < 0 {
			continue
		}
		hunk := &files[currentFile].Hunks[currentHunk]
		if line == `\ No newline at end of file` {
			hunk.Lines = append(hunk.Lines, renderLine{Kind: renderLineNote, Text: line})
			continue
		}
		if line == "" {
			continue
		}
		switch line[0] {
		case '+':
			hunk.Lines = append(hunk.Lines, renderLine{Kind: renderLineAdd, Text: line})
		case '-':
			hunk.Lines = append(hunk.Lines, renderLine{Kind: renderLineDelete, Text: line})
		case ' ':
			hunk.Lines = append(hunk.Lines, renderLine{Kind: renderLineContext, Text: line})
		default:
			hunk.Lines = append(hunk.Lines, renderLine{Kind: renderLineNote, Text: line})
		}
	}

	return files
}

func colorRenderLine(line renderLine, width, hScroll int) string {
	text := padDisplay(cutDisplay(line.Text, hScroll, width), width)
	switch line.Kind {
	case renderLineAdd:
		return styleAddLine.Width(width).Render(text)
	case renderLineDelete:
		return styleDelLine.Width(width).Render(text)
	case renderLineNote:
		return styleMuted.Render(text)
	default:
		return text
	}
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
