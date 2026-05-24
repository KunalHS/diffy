package main

import (
	"fmt"
	"strings"
)

func FormatSummary(view ViewData) string {
	if len(view.Files) == 0 {
		if strings.TrimSpace(view.Message) != "" {
			return view.Message + "\n"
		}
		return "0 files changed, +0 -0\n"
	}
	add, del := totalCounts(view.Files)
	var b strings.Builder
	fmt.Fprintf(&b, "%d files changed, +%d -%d\n\n", len(view.Files), add, del)
	width := maxPathWidth(view.Files)
	for _, file := range view.Files {
		if file.Binary {
			fmt.Fprintf(&b, "%-*s  binary\n", width, file.Path)
			continue
		}
		fmt.Fprintf(&b, "%-*s  +%d -%d\n", width, file.Path, file.Add, file.Del)
	}
	return b.String()
}

func totalCounts(files []FileStat) (int, int) {
	add, del := 0, 0
	for _, file := range files {
		add += file.Add
		del += file.Del
	}
	return add, del
}

func maxPathWidth(files []FileStat) int {
	width := 0
	for _, file := range files {
		if len(file.Path) > width {
			width = len(file.Path)
		}
	}
	if width > 72 {
		return 72
	}
	return width
}
