
package helpers

import (
	"io"
	"os"
	"path/filepath"
)

func SaveUploadedFile(
	src io.Reader,
	uploadDir string,
	extension string,
) (string, error) {

	if extension == "" {
		extension = ".mp4"
	}

	if err := EnsureDirectory(uploadDir); err != nil {
		return "", err
	}

	dst, err := os.CreateTemp(
		uploadDir,
		"upload-*"+extension,
	)
	if err != nil {
		return "", err
	}

	path := dst.Name()

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()

	if copyErr != nil {
		_ = os.Remove(path)
		return "", copyErr
	}

	if closeErr != nil {
		_ = os.Remove(path)
		return "", closeErr
	}

	return filepath.Abs(path)
}
