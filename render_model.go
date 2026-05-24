package main

type RenderRow struct {
	Row           DiffRow
	BlockPosition BlockPosition
	OldInline     []InlineSpan
	NewInline     []InlineSpan
}

type BlockPosition int

const (
	BlockNone BlockPosition = iota
	BlockStart
	BlockMiddle
	BlockEnd
	BlockSingle
)

type InlineSpan struct {
	Start int
	End   int
	Kind  InlineSpanKind
}

type InlineSpanKind int

const (
	InlineChanged InlineSpanKind = iota
)

func PrepareRenderRows(rows []DiffRow) []RenderRow {
	prepared := make([]RenderRow, len(rows))
	for i, row := range rows {
		prepared[i] = RenderRow{Row: row}
		if row.Kind == RowModify {
			prepared[i].OldInline, prepared[i].NewInline = ChangedSpans(row.OldText, row.NewText)
		}
	}

	for start := 0; start < len(prepared); {
		if !isChangedRow(prepared[start].Row.Kind) {
			start++
			continue
		}
		end := start
		for end < len(prepared) && isChangedRow(prepared[end].Row.Kind) {
			end++
		}
		for i := start; i < end; i++ {
			prepared[i].BlockPosition = blockPosition(i, start, end)
		}
		start = end
	}
	return prepared
}

func ChangedSpans(oldText, newText string) ([]InlineSpan, []InlineSpan) {
	oldRunes := []rune(oldText)
	newRunes := []rune(newText)
	if string(oldRunes) == string(newRunes) {
		return nil, nil
	}

	prefix := 0
	for prefix < len(oldRunes) && prefix < len(newRunes) && oldRunes[prefix] == newRunes[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(oldRunes)-prefix && suffix < len(newRunes)-prefix &&
		oldRunes[len(oldRunes)-1-suffix] == newRunes[len(newRunes)-1-suffix] {
		suffix++
	}

	oldStart, oldEnd := prefix, len(oldRunes)-suffix
	newStart, newEnd := prefix, len(newRunes)-suffix
	return spanIfNotEmpty(oldStart, oldEnd), spanIfNotEmpty(newStart, newEnd)
}

func spanIfNotEmpty(start, end int) []InlineSpan {
	if end <= start {
		return nil
	}
	return []InlineSpan{{Start: start, End: end, Kind: InlineChanged}}
}

func isChangedRow(kind DiffRowKind) bool {
	return kind == RowAdd || kind == RowDelete || kind == RowModify
}

func blockPosition(index, start, end int) BlockPosition {
	if end-start == 1 {
		return BlockSingle
	}
	switch index {
	case start:
		return BlockStart
	case end - 1:
		return BlockEnd
	default:
		return BlockMiddle
	}
}
