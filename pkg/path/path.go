package path

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindRoot(startDir, targetName string, isDir bool) (string, error) {
	dir := startDir

	for {
		fullPath := filepath.Join(dir, targetName)
		if info, err := os.Stat(fullPath); err == nil {
			if isDir && info.IsDir() {
				return dir, nil
			} else if !isDir && !info.IsDir() {
				return dir, nil
			}
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	return "", fmt.Errorf("could not find %s starting from %s", targetName, startDir)
}
