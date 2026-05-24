package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func LoadConfig() Config {
	cfg := Config{Layout: LayoutSplit, Sidebar: true}
	cfg.Path = configPath()
	cfg.StatePath = statePathForConfig(cfg.Path)

	file, err := os.Open(cfg.Path)
	if err != nil {
		return cfg
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			key, value, ok = strings.Cut(line, "=")
		}
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch key {
		case "layout":
			switch strings.ToLower(value) {
			case "unified":
				cfg.Layout = LayoutUnified
				cfg.LayoutSet = true
			case "split":
				cfg.Layout = LayoutSplit
				cfg.LayoutSet = true
			}
		case "sidebar":
			if parsed, err := strconv.ParseBool(strings.ToLower(value)); err == nil {
				cfg.Sidebar = parsed
				cfg.SidebarSet = true
			}
		}
	}

	return cfg
}

func LoadState(path string) DiffyState {
	state := DiffyState{Path: path}
	file, err := os.Open(path)
	if err != nil {
		return state
	}
	defer file.Close()

	var raw struct {
		Layout  *Layout `json:"layout,omitempty"`
		Sidebar *bool   `json:"sidebar,omitempty"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return state
	}
	if raw.Layout != nil {
		switch *raw.Layout {
		case LayoutSplit, LayoutUnified:
			state.Layout = *raw.Layout
			state.LayoutSet = true
		}
	}
	if raw.Sidebar != nil {
		state.Sidebar = *raw.Sidebar
		state.SidebarSet = true
	}
	return state
}

func SaveState(state DiffyState) error {
	if state.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(state.Path), 0o755); err != nil {
		return err
	}
	raw := struct {
		Layout  *Layout `json:"layout,omitempty"`
		Sidebar *bool   `json:"sidebar,omitempty"`
	}{}
	if state.LayoutSet {
		raw.Layout = &state.Layout
	}
	if state.SidebarSet {
		raw.Sidebar = &state.Sidebar
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(state.Path, data, 0o644)
}

func ResolveConfig(cfg Config, state DiffyState) Config {
	if !cfg.LayoutSet && state.LayoutSet {
		cfg.Layout = state.Layout
	}
	if !cfg.SidebarSet && state.SidebarSet {
		cfg.Sidebar = state.Sidebar
	}
	return cfg
}

func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "diffy", "config.yaml")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config", "diffy", "config.yaml")
	}
	return filepath.Join(".config", "diffy", "config.yaml")
}

func statePathForConfig(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "state.json")
}
