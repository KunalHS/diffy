package main

import "testing"

func TestChangedSpans(t *testing.T) {
	tests := []struct {
		name     string
		oldText  string
		newText  string
		oldSpans []InlineSpan
		newSpans []InlineSpan
	}{
		{
			name:     "same text",
			oldText:  "same",
			newText:  "same",
			oldSpans: nil,
			newSpans: nil,
		},
		{
			name:     "suffix changed",
			oldText:  "return oldValue",
			newText:  "return newValue",
			oldSpans: []InlineSpan{{Start: 7, End: 10, Kind: InlineChanged}},
			newSpans: []InlineSpan{{Start: 7, End: 10, Kind: InlineChanged}},
		},
		{
			name:     "prefix changed",
			oldText:  "oldValue.done()",
			newText:  "newValue.done()",
			oldSpans: []InlineSpan{{Start: 0, End: 3, Kind: InlineChanged}},
			newSpans: []InlineSpan{{Start: 0, End: 3, Kind: InlineChanged}},
		},
		{
			name:     "middle token changed",
			oldText:  "import com.ontic.fwk.common.IdLabel;",
			newText:  "import com.ontic.core.signals.topic.TopicUtils;",
			oldSpans: []InlineSpan{{Start: 17, End: 35, Kind: InlineChanged}},
			newSpans: []InlineSpan{{Start: 17, End: 46, Kind: InlineChanged}},
		},
		{
			name:     "entire line changed",
			oldText:  "abc",
			newText:  "xyz",
			oldSpans: []InlineSpan{{Start: 0, End: 3, Kind: InlineChanged}},
			newSpans: []InlineSpan{{Start: 0, End: 3, Kind: InlineChanged}},
		},
		{
			name:     "insertion only",
			oldText:  "value",
			newText:  "valueExtra",
			oldSpans: nil,
			newSpans: []InlineSpan{{Start: 5, End: 10, Kind: InlineChanged}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSpans, newSpans := ChangedSpans(tt.oldText, tt.newText)
			assertSpans(t, oldSpans, tt.oldSpans)
			assertSpans(t, newSpans, tt.newSpans)
		})
	}
}

func TestPrepareRenderRowsAddsBlockPositionsAndInlineSpans(t *testing.T) {
	rows := []DiffRow{
		{Kind: RowContext, OldText: "same", NewText: "same"},
		{Kind: RowAdd, NewText: "added 1"},
		{Kind: RowAdd, NewText: "added 2"},
		{Kind: RowModify, OldText: "return oldValue", NewText: "return newValue"},
		{Kind: RowContext, OldText: "same", NewText: "same"},
		{Kind: RowDelete, OldText: "deleted"},
	}

	prepared := PrepareRenderRows(rows)
	wantPositions := []BlockPosition{BlockNone, BlockStart, BlockMiddle, BlockEnd, BlockNone, BlockSingle}
	for i, want := range wantPositions {
		if prepared[i].BlockPosition != want {
			t.Fatalf("row %d block position = %v, want %v", i, prepared[i].BlockPosition, want)
		}
	}
	if len(prepared[3].OldInline) == 0 || len(prepared[3].NewInline) == 0 {
		t.Fatalf("modify row missing inline spans: %#v", prepared[3])
	}
	if len(prepared[0].OldInline) != 0 || len(prepared[0].NewInline) != 0 {
		t.Fatalf("context row received inline spans: %#v", prepared[0])
	}
}

func assertSpans(t *testing.T, got, want []InlineSpan) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("spans = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("spans = %#v, want %#v", got, want)
		}
	}
}
