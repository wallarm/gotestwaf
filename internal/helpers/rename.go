package helpers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Move moves a file from a source path to a destination path.
// This must be used across the codebase for compatibility with Docker volumes
// and Golang (fixes Invalid cross-device link when using [os.Rename])
func Move(sourcePath, destPath string) error {
	sourceAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return err
	}

	destAbs, err := filepath.Abs(destPath)
	if err != nil {
		return err
	}

	if sourceAbs == destAbs {
		return nil
	}

	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	destDir := filepath.Dir(destPath)
	_, err = os.Stat(destDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			inputFile.Close()
			return err
		}
	}

	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}

	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	outputFile.Close()

	if err != nil {
		if errRem := os.Remove(destPath); errRem != nil {
			return fmt.Errorf(
				"unable to os.Remove error: %s after io.Copy error: %s",
				errRem,
				err,
			)
		}

		return err
	}

	return os.Remove(sourcePath)
}
