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
