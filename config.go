package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func LoadConfig() Config {
	cfg := Config{Layout: LayoutSplit}
	cfg.Path = configPath()

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
		if key == "layout" {
			switch strings.ToLower(value) {
			case "unified":
				cfg.Layout = LayoutUnified
			case "split":
				cfg.Layout = LayoutSplit
			}
		}
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
