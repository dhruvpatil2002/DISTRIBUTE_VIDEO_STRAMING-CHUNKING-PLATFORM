
package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDirectory(path string) error {

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return nil
}

func AbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}
