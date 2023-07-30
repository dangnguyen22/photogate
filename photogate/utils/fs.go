package utils

import (
	"io/fs"
	"path/filepath"
	"regexp"
)

func ScanFileMatch(targetFs fs.FS, rootDir string, r *regexp.Regexp, cb func(string, string, []byte) error) error {
	dirs, err := fs.ReadDir(targetFs, rootDir)
	if err != nil {
		return err
	}
	for _, de := range dirs {
		if de.IsDir() || !r.MatchString(de.Name()) {
			continue
		}

		path := filepath.Join(rootDir, de.Name())
		b, err := fs.ReadFile(targetFs, path)
		if err != nil {
			return err
		}

		if err = cb(de.Name(), path, b); err != nil {
			return err
		}
	}

	return nil
}
