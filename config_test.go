package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigSupportsLayoutAndSidebar(t *testing.T) {
	configDir := diffyConfigDir(t)
	writeConfigTestFile(t, filepath.Join(configDir, "config.yaml"), "layout: unified\nsidebar: false\n")

	cfg := LoadConfig()
	if cfg.Layout != LayoutUnified || !cfg.LayoutSet {
		t.Fatalf("layout = %q set=%t, want unified set", cfg.Layout, cfg.LayoutSet)
	}
	if cfg.Sidebar || !cfg.SidebarSet {
		t.Fatalf("sidebar = %t set=%t, want false set", cfg.Sidebar, cfg.SidebarSet)
	}
	if cfg.StatePath != filepath.Join(configDir, "state.json") {
		t.Fatalf("state path = %q, want state file beside config", cfg.StatePath)
	}
}

func TestResolveConfigUsesStateWhenConfigIsSilent(t *testing.T) {
	configDir := diffyConfigDir(t)
	statePath := filepath.Join(configDir, "state.json")
	writeConfigTestFile(t, statePath, "{\n  \"layout\": \"unified\",\n  \"sidebar\": false\n}\n")

	cfg := LoadConfig()
	state := LoadState(cfg.StatePath)
	resolved := ResolveConfig(cfg, state)

	if resolved.Layout != LayoutUnified {
		t.Fatalf("layout = %q, want state layout unified", resolved.Layout)
	}
	if resolved.Sidebar {
		t.Fatalf("sidebar = true, want state sidebar false")
	}
}

func TestResolveConfigPrefersConfigOverState(t *testing.T) {
	configDir := diffyConfigDir(t)
	writeConfigTestFile(t, filepath.Join(configDir, "config.yaml"), "layout: split\nsidebar: true\n")
	writeConfigTestFile(t, filepath.Join(configDir, "state.json"), "{\n  \"layout\": \"unified\",\n  \"sidebar\": false\n}\n")

	cfg := LoadConfig()
	state := LoadState(cfg.StatePath)
	resolved := ResolveConfig(cfg, state)

	if resolved.Layout != LayoutSplit {
		t.Fatalf("layout = %q, want config layout split", resolved.Layout)
	}
	if !resolved.Sidebar {
		t.Fatalf("sidebar = false, want config sidebar true")
	}
}

func TestSaveStateWritesLayoutAndSidebar(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state", "state.json")
	err := SaveState(DiffyState{
		Path:       statePath,
		Layout:     LayoutUnified,
		LayoutSet:  true,
		Sidebar:    false,
		SidebarSet: true,
	})
	if err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	state := LoadState(statePath)
	if state.Layout != LayoutUnified || !state.LayoutSet {
		t.Fatalf("layout = %q set=%t, want unified set", state.Layout, state.LayoutSet)
	}
	if state.Sidebar || !state.SidebarSet {
		t.Fatalf("sidebar = %t set=%t, want false set", state.Sidebar, state.SidebarSet)
	}
}

func diffyConfigDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	configDir := filepath.Join(root, "diffy")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	return configDir
}

func writeConfigTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
