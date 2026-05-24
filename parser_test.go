package main

import "testing"

func TestParseApprovedAliases(t *testing.T) {
	cfg := Config{Layout: LayoutSplit}

	tests := []struct {
		name   string
		args   []string
		kind   CommandKind
		source string
		target string
	}{
		{name: "bare local", args: nil, kind: KindLocal},
		{name: "local command", args: []string{"local"}, kind: KindLocal},
		{name: "bare target compare", args: []string{"live"}, kind: KindCompare, target: "live"},
		{name: "bare source target compare", args: []string{"b1", "b2"}, kind: KindCompare, source: "b1", target: "b2"},
		{name: "review target", args: []string{"review", "main"}, kind: KindCompare, target: "main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.args, cfg)
			if err != nil {
				t.Fatalf("ParseCommand returned error: %v", err)
			}
			if cmd.Kind != tt.kind {
				t.Fatalf("kind = %s, want %s", cmd.Kind, tt.kind)
			}
			if cmd.Source != tt.source {
				t.Fatalf("source = %q, want %q", cmd.Source, tt.source)
			}
			if cmd.Target != tt.target {
				t.Fatalf("target = %q, want %q", cmd.Target, tt.target)
			}
		})
	}
}

func TestParseRejectsStackedAliases(t *testing.T) {
	cfg := Config{Layout: LayoutSplit}
	bad := [][]string{
		{"compare", "review", "live"},
		{"review", "compare", "live"},
		{"compare", "live", "--file", "a.go", "--path", "a.go"},
		{"compare", "live", "--raw", "--no-tui"},
	}
	for _, args := range bad {
		if _, err := ParseCommand(args, cfg); err == nil {
			t.Fatalf("ParseCommand(%v) succeeded, want error", args)
		}
	}
}

func TestParseSharedOptions(t *testing.T) {
	cfg := Config{Layout: LayoutSplit}
	cmd, err := ParseCommand([]string{"recent", "3", "--path", "a.go", "--summary", "--unified"}, cfg)
	if err != nil {
		t.Fatalf("ParseCommand returned error: %v", err)
	}
	if cmd.Kind != KindRecent || cmd.Count != 3 {
		t.Fatalf("parsed command = %#v", cmd)
	}
	if !cmd.Options.Summary || !cmd.Options.FileFlag || cmd.Options.FilePath != "a.go" {
		t.Fatalf("options = %#v", cmd.Options)
	}
	if cmd.Options.Layout != LayoutUnified {
		t.Fatalf("layout = %s, want unified", cmd.Options.Layout)
	}
}

func TestParseFileComparePickerForms(t *testing.T) {
	cfg := Config{Layout: LayoutSplit}
	tests := []struct {
		name    string
		args    []string
		path    string
		ref     string
		fromRef string
		toRef   string
	}{
		{name: "picker no refs", args: []string{"file"}},
		{name: "picker one ref", args: []string{"file", "main"}, ref: "main"},
		{name: "path option one ref", args: []string{"file", "--file", "src/App.ts", "main"}, path: "src/App.ts", ref: "main"},
		{name: "path option two refs", args: []string{"file", "--path", "src/App.ts", "main", "HEAD"}, path: "src/App.ts", fromRef: "main", toRef: "HEAD"},
		{name: "legacy path first", args: []string{"file", "src/App.ts", "main"}, path: "src/App.ts", ref: "main"},
		{name: "legacy path first two refs", args: []string{"file", "src/App.ts", "main", "HEAD"}, path: "src/App.ts", fromRef: "main", toRef: "HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.args, cfg)
			if err != nil {
				t.Fatalf("ParseCommand returned error: %v", err)
			}
			if cmd.Kind != KindFile {
				t.Fatalf("kind = %s, want file", cmd.Kind)
			}
			if cmd.Path != tt.path || cmd.Ref != tt.ref || cmd.FromRef != tt.fromRef || cmd.ToRef != tt.toRef {
				t.Fatalf("cmd = %#v", cmd)
			}
		})
	}
}
