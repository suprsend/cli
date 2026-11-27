package utils

import (
	"os"
	"strings"
)

// IsHidden reports whether a file or directory name starts with a dot. No
// asset name does (slugs start with a letter or digit, event directories
// with a letter or $, translation files are locale codes), so the CLI skips
// these: its own .suprsend-state.json, and strays such as .DS_Store, .git,
// .idea or macOS ._en.json files.
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// ReadDir is os.ReadDir without hidden entries (see IsHidden). Use it
// wherever the CLI lists a directory of assets.
func ReadDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	visible := entries[:0]
	for _, e := range entries {
		if !IsHidden(e.Name()) {
			visible = append(visible, e)
		}
	}
	return visible, err
}
