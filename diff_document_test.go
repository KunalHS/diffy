package main

import "testing"

func TestParseDiffDocumentAssignsLineNumbersAndHidesHunks(t *testing.T) {
	diff := `diff --git a/app.txt b/app.txt
index 1111111..2222222 100644
--- a/app.txt
+++ b/app.txt
@@ -10,4 +10,5 @@ func main() {
 context one
-old value
+new value
 context two
+added value`

	doc := ParseDiffDocument(diff)
	if len(doc.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(doc.Files))
	}
	rows := doc.Files[0].Rows
	if len(rows) != 4 {
		t.Fatalf("rows = %#v, want 4 rows", rows)
	}
	if rows[0].Kind != RowContext || rows[0].OldLine != 10 || rows[0].NewLine != 10 {
		t.Fatalf("first row = %#v, want context line 10/10", rows[0])
	}
	if rows[1].Kind != RowModify || rows[1].OldLine != 11 || rows[1].NewLine != 11 || rows[1].OldText != "old value" || rows[1].NewText != "new value" {
		t.Fatalf("modify row = %#v", rows[1])
	}
	if rows[3].Kind != RowAdd || rows[3].NewLine != 13 || rows[3].NewText != "added value" {
		t.Fatalf("add row = %#v", rows[3])
	}
	for _, row := range rows {
		if row.OldText == "@@ -10,4 +10,5 @@ func main() {" || row.NewText == "@@ -10,4 +10,5 @@ func main() {" {
			t.Fatalf("hunk header leaked into display rows: %#v", row)
		}
	}
}

func TestParseDiffDocumentKeepsPatchLikeContentInsideHunks(t *testing.T) {
	diff := `diff --git a/text.txt b/text.txt
--- a/text.txt
+++ b/text.txt
@@ -1,1 +1,4 @@
 line one
+diff --git is content here
+index 0000000 is content here
+--- /dev/null is content here`

	doc := ParseDiffDocument(diff)
	rows := doc.Files[0].Rows
	if rows[1].Kind != RowAdd || rows[1].NewText != "diff --git is content here" {
		t.Fatalf("patch-like add row = %#v", rows[1])
	}
	if rows[3].Kind != RowAdd || rows[3].NewText != "--- /dev/null is content here" {
		t.Fatalf("patch-like add row = %#v", rows[3])
	}
}

func TestParseDiffDocumentMarksNewAndDeletedFiles(t *testing.T) {
	newFile := ParseDiffDocument(`diff --git a/new.txt b/new.txt
new file mode 100644
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,1 @@
+hello`)
	if len(newFile.Files) != 1 || !newFile.Files[0].IsNew || newFile.Files[0].NewPath != "new.txt" {
		t.Fatalf("new file = %#v", newFile.Files)
	}

	deletedFile := ParseDiffDocument(`diff --git a/old.txt b/old.txt
deleted file mode 100644
--- a/old.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-bye`)
	if len(deletedFile.Files) != 1 || !deletedFile.Files[0].IsDeleted || deletedFile.Files[0].OldPath != "old.txt" {
		t.Fatalf("deleted file = %#v", deletedFile.Files)
	}
}

func TestVisibleRowsCollapseLargeContextRun(t *testing.T) {
	file := DiffFile{Rows: []DiffRow{{Kind: RowDelete, OldLine: 1, OldText: "before"}}}
	for i := 0; i < 9; i++ {
		line := i + 2
		file.Rows = append(file.Rows, DiffRow{Kind: RowContext, OldLine: line, NewLine: line, OldText: "same", NewText: "same"})
	}
	file.Rows = append(file.Rows, DiffRow{Kind: RowAdd, NewLine: 11, NewText: "after"})

	rows := VisibleRowsForFile(file, 0, DiffRenderOptions{ContextLines: 2})
	found := false
	for _, row := range rows {
		if row.Kind == RowCollapsed {
			found = true
			if row.Collapsed.Count != 5 {
				t.Fatalf("collapsed count = %d, want 5", row.Collapsed.Count)
			}
		}
	}
	if !found {
		t.Fatalf("collapsed row not found: %#v", rows)
	}

	expanded := VisibleRowsForFile(file, 0, DiffRenderOptions{
		ContextLines: 2,
		Expanded:     map[string]bool{"file:0:context:1:10": true},
	})
	if len(expanded) != len(file.Rows) {
		t.Fatalf("expanded rows = %d, want %d", len(expanded), len(file.Rows))
	}
}
