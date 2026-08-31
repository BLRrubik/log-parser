package scanner

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func ScanLogDirectory(dirPath string) ([]string, error) {
	var logFiles []string

	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".log") {
			logFiles = append(logFiles, path)
		}
		return nil
	})

	return logFiles, nil
}
