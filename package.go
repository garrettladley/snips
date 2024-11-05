package snips

import (
	"os"
	"path/filepath"
	"strings"
)

func PackageName(dir string) (name string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".templ") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "package ") {
					name = strings.TrimSpace(strings.TrimPrefix(line, "package"))
					return filepath.SkipAll // stop walking, we found a package name
				}
			}
		}
		return nil
	})

	if err != nil || name == "" {
		return fallback(dir)
	}

	return name
}

func fallback(dir string) (name string) {
	var (
		parts = strings.Split(filepath.ToSlash(dir), "/")
		n     = len(parts)
	)

	if n > 1 {
		return parts[n-1]
	}

	return "main"
}
