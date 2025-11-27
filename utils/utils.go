package utils

import (
	"io/fs"
	"path/filepath"
)

// GetFilePaths gets all file paths in the specified directory
func GetFilePaths(rootDir string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			paths = append(paths, absPath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return paths, nil
}
