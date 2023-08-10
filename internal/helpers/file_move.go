package helpers

import (
	"fmt"
	"io"
	"os"
)

// FileMove moves a file from a source path to a destination path.
// This must be used across the codebase for compatibility with Docker volumes
// and Golang (fixes Invalid cross-device link when using [os.Rename])
func FileMove(sourcePath, destPath string) error {
	// check the source file is exist
	sourceFileStat, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// check the destination file
	destFileStat, err := os.Stat(destPath)
	if err == nil {
		// return error if the destination file is the same file as source one
		if sourcePath == destPath || os.SameFile(sourceFileStat, destFileStat) {
			return fmt.Errorf("files %s and %s are the same", sourcePath, destPath)
		}
	}

	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
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
