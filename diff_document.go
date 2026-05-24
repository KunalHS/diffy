package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type DiffDocument struct {
	Files []DiffFile
}

type DiffFile struct {
	OldPath   string
	NewPath   string
	IsNew     bool
	IsDeleted bool
	Rows      []DiffRow
}

type DiffRow struct {
	Kind      DiffRowKind
	OldLine   int
	NewLine   int
	OldText   string
	NewText   string
	Note      string
	Collapsed *CollapsedContext
}

type DiffRowKind int

const (
	RowContext DiffRowKind = iota
	RowAdd
	RowDelete
	RowModify
	RowNote
	RowCollapsed
)

type CollapsedContext struct {
	Key   string
	Count int
	Rows  []DiffRow
}

type DiffRenderOptions struct {
	Expanded     map[string]bool
	ExpandAll    bool
	ContextLines int
}

type CollapsedContextTarget struct {
	Key  string
	Line int
}

var hunkRangePattern = regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)

func ParseDiffDocument(diff string) DiffDocument {
	var doc DiffDocument
	var current *DiffFile
	var oldLine, newLine int
	var inHunk bool
	var pendingDeletes []DiffRow

	flushDeletes := func() {
		if current == nil || len(pendingDeletes) == 0 {
			pendingDeletes = nil
			return
		}
		current.Rows = append(current.Rows, pendingDeletes...)
		pendingDeletes = nil
	}

	ensureFile := func() *DiffFile {
		if current == nil {
			doc.Files = append(doc.Files, DiffFile{})
			current = &doc.Files[len(doc.Files)-1]
		}
		return current
	}

	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			flushDeletes()
			oldPath, newPath := parseDiffGitPaths(line)
			doc.Files = append(doc.Files, DiffFile{OldPath: oldPath, NewPath: newPath})
			current = &doc.Files[len(doc.Files)-1]
			inHunk = false
			continue
		}

		if strings.HasPrefix(line, "@@") {
			flushDeletes()
			file := ensureFile()
			parsedOld, parsedNew, ok := parseHunkStart(line)
			if !ok {
				file.Rows = append(file.Rows, DiffRow{Kind: RowNote, Note: line})
				continue
			}
			oldLine = parsedOld
			newLine = parsedNew
			inHunk = true
			continue
		}

		if current != nil && !inHunk {
			switch {
			case strings.HasPrefix(line, "--- "):
				path := parseDiffPath(strings.TrimSpace(strings.TrimPrefix(line, "--- ")))
				current.OldPath = path
				current.IsNew = path == "/dev/null"
			case strings.HasPrefix(line, "+++ "):
				path := parseDiffPath(strings.TrimSpace(strings.TrimPrefix(line, "+++ ")))
				current.NewPath = path
				current.IsDeleted = path == "/dev/null"
			}
			continue
		}

		if !inHunk {
			continue
		}

		file := ensureFile()
		if line == `\ No newline at end of file` {
			flushDeletes()
			file.Rows = append(file.Rows, DiffRow{Kind: RowNote, Note: "No newline at end of file"})
			continue
		}
		if line == "" {
			continue
		}

		switch line[0] {
		case ' ':
			flushDeletes()
			text := strings.TrimPrefix(line, " ")
			file.Rows = append(file.Rows, DiffRow{
				Kind:    RowContext,
				OldLine: oldLine,
				NewLine: newLine,
				OldText: text,
				NewText: text,
			})
			oldLine++
			newLine++
		case '-':
			pendingDeletes = append(pendingDeletes, DiffRow{
				Kind:    RowDelete,
				OldLine: oldLine,
				OldText: strings.TrimPrefix(line, "-"),
			})
			oldLine++
		case '+':
			text := strings.TrimPrefix(line, "+")
			if len(pendingDeletes) > 0 {
				deleted := pendingDeletes[0]
				pendingDeletes = pendingDeletes[1:]
				file.Rows = append(file.Rows, DiffRow{
					Kind:    RowModify,
					OldLine: deleted.OldLine,
					NewLine: newLine,
					OldText: deleted.OldText,
					NewText: text,
				})
			} else {
				file.Rows = append(file.Rows, DiffRow{
					Kind:    RowAdd,
					NewLine: newLine,
					NewText: text,
				})
			}
			newLine++
		default:
			flushDeletes()
			file.Rows = append(file.Rows, DiffRow{Kind: RowNote, Note: line})
		}
	}
	flushDeletes()

	return doc
}

func VisibleRowsForFile(file DiffFile, fileIndex int, opts DiffRenderOptions) []DiffRow {
	contextLines := opts.ContextLines
	if contextLines <= 0 {
		contextLines = 3
	}
	var visible []DiffRow
	for start := 0; start < len(file.Rows); {
		row := file.Rows[start]
		if row.Kind != RowContext {
			visible = append(visible, row)
			start++
			continue
		}

		end := start
		for end < len(file.Rows) && file.Rows[end].Kind == RowContext {
			end++
		}
		run := file.Rows[start:end]
		visible = append(visible, collapseContextRun(run, fileIndex, start, end, start > 0, end < len(file.Rows), opts, contextLines)...)
		start = end
	}
	return visible
}

func CollapsedContextKeys(diff string, opts DiffRenderOptions) []string {
	var keys []string
	for _, target := range CollapsedContextTargets(diff, LayoutSplit, opts) {
		keys = append(keys, target.Key)
	}
	return keys
}

func CollapsedContextTargets(diff string, layout Layout, opts DiffRenderOptions) []CollapsedContextTarget {
	doc := ParseDiffDocument(diff)
	var targets []CollapsedContextTarget
	line := 0
	for fileIndex, file := range doc.Files {
		if fileIndex > 0 {
			line++
		}
		line++
		for _, row := range VisibleRowsForFile(file, fileIndex, opts) {
			if row.Kind == RowCollapsed && row.Collapsed != nil {
				targets = append(targets, CollapsedContextTarget{Key: row.Collapsed.Key, Line: line})
			}
			line += renderedRowHeight(row, layout)
		}
	}
	return targets
}

func renderedRowHeight(row DiffRow, layout Layout) int {
	if layout == LayoutUnified && row.Kind == RowModify {
		return 2
	}
	return 1
}

func collapseContextRun(run []DiffRow, fileIndex, start, end int, hasBefore, hasAfter bool, opts DiffRenderOptions, contextLines int) []DiffRow {
	if len(run) <= contextLines*2+1 {
		return append([]DiffRow(nil), run...)
	}
	key := fmt.Sprintf("file:%d:context:%d:%d", fileIndex, start, end)
	if opts.ExpandAll || opts.Expanded[key] {
		return append([]DiffRow(nil), run...)
	}

	switch {
	case hasBefore && hasAfter:
		hiddenStart := contextLines
		hiddenEnd := len(run) - contextLines
		return collapsedRunRows(run, key, hiddenStart, hiddenEnd)
	case hasBefore:
		hiddenStart := contextLines
		hiddenEnd := len(run)
		return collapsedRunRows(run, key, hiddenStart, hiddenEnd)
	case hasAfter:
		hiddenStart := 0
		hiddenEnd := len(run) - contextLines
		return collapsedRunRows(run, key, hiddenStart, hiddenEnd)
	default:
		return append([]DiffRow(nil), run...)
	}
}

func collapsedRunRows(run []DiffRow, key string, hiddenStart, hiddenEnd int) []DiffRow {
	if hiddenStart < 0 {
		hiddenStart = 0
	}
	if hiddenEnd > len(run) {
		hiddenEnd = len(run)
	}
	if hiddenEnd <= hiddenStart {
		return append([]DiffRow(nil), run...)
	}

	rows := append([]DiffRow(nil), run[:hiddenStart]...)
	hidden := append([]DiffRow(nil), run[hiddenStart:hiddenEnd]...)
	rows = append(rows, DiffRow{
		Kind: RowCollapsed,
		Collapsed: &CollapsedContext{
			Key:   key,
			Count: len(hidden),
			Rows:  hidden,
		},
	})
	rows = append(rows, run[hiddenEnd:]...)
	return rows
}

func parseHunkStart(line string) (int, int, bool) {
	matches := hunkRangePattern.FindStringSubmatch(line)
	if matches == nil {
		return 0, 0, false
	}
	oldStart, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, false
	}
	newStart, err := strconv.Atoi(matches[3])
	if err != nil {
		return 0, 0, false
	}
	return oldStart, newStart, true
}

func parseDiffGitPaths(line string) (string, string) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return "", ""
	}
	return parseDiffPath(parts[2]), parseDiffPath(parts[3])
}

func parseDiffPath(path string) string {
	path = strings.Trim(path, `"`)
	if path == "/dev/null" {
		return path
	}
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	return path
}
