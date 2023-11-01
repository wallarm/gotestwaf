package helpers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var testDir string

func createTestFile(dst string, content string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	var err error
	testDir, err = os.MkdirTemp("", "gtw_test_dir")
	if err != nil {
		fmt.Println("Couldn't create directory for test content:", err.Error())
		os.Exit(1)
	}

	exitVal := m.Run()

	err = os.RemoveAll(testDir)
	if err != nil {
		fmt.Println("Couldn't remove directory for test content:", err.Error())
		os.Exit(1)
	}

	os.Exit(exitVal)
}

func TestFileMoveSourceFileNotExist(t *testing.T) {
	srcFile := filepath.Join(testDir, "gtw_test_file_not_exist.txt")
	dstFile := filepath.Join(testDir, "gtw_test_file_not_exist_dst.txt")

	err := FileMove(srcFile, dstFile)
	if err == nil {
		t.Errorf("try to move file that is not existed, err must not be nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err must be os.ErrNotExist")
	}
}

func TestFileMoveSourceDestinationTheSamePath(t *testing.T) {
	srcFile := filepath.Join(testDir, "gtw_test_file.txt")
	dstFile := srcFile

	err := createTestFile(srcFile, "test")
	if err != nil {
		t.Errorf("couldn't create test file: %v", err)
	}
	defer func() {
		err = os.Remove(srcFile)
		if err != nil {
			t.Logf("couldn't remove test file: %v", err)
		}
	}()

	err = FileMove(srcFile, dstFile)
	if err == nil {
		t.Errorf("the dstFile is the same as the srcFile, couln't move srcFile to dstFile, err must not be nil")
	}
}

func TestFileMoveSourceDestinationTheSameFile(t *testing.T) {
	srcFile := filepath.Join(testDir, "gtw_test_file.txt")
	dstFile := filepath.Join(testDir, "gtw_test_file_link.txt")

	err := createTestFile(srcFile, "test")
	if err != nil {
		t.Errorf("couldn't create test file: %v", err)
	}
	defer func() {
		err = os.Remove(srcFile)
		if err != nil {
			t.Logf("couldn't remove test file: %v", err)
		}
	}()

	err = os.Link(srcFile, dstFile)
	if err != nil {
		t.Errorf("couldn't create link for test file: %v", err)
	}
	defer func() {
		err = os.Remove(dstFile)
		if err != nil {
			t.Logf("couldn't remove link for test file: %v", err)
		}
	}()

	err = FileMove(srcFile, dstFile)
	if err == nil {
		t.Errorf("the dstFile is the link to the srcFile, couln't move srcFile to dstFile, err must not be nil")
	}
}

func TestFileMove(t *testing.T) {
	srcFile := filepath.Join(testDir, "gtw_test_file_1.txt")
	dstFile := filepath.Join(testDir, "gtw_test_file_2.txt")
	fileContent := "test1"

	err := createTestFile(srcFile, fileContent)
	if err != nil {
		t.Errorf("couldn't create test file: %v", err)
	}

	// cleanup
	defer func() {
		os.Remove(srcFile)
		os.Remove(dstFile)
	}()

	err = FileMove(srcFile, dstFile)
	if err != nil {
		t.Errorf("err is not nil: %v", err)
	}

	_, err = os.Stat(srcFile)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file %s must not exist", srcFile)
	}

	_, err = os.Stat(dstFile)
	if err != nil {
		t.Errorf("file %s must exist", srcFile)
	}

	dstFileContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Errorf("couldn't reade dstFile content: %v", err)
	}

	if fileContent != string(dstFileContent) {
		t.Errorf("dstFile content is not the same as srcFile content: '%s' != '%s'", dstFileContent, fileContent)
	}
}

func TestFileMoveReplace(t *testing.T) {
	srcFile := filepath.Join(testDir, "gtw_test_file_1.txt")
	dstFile := filepath.Join(testDir, "gtw_test_file_2.txt")
	srcFileContent := "test1"
	dstFileContent := "test2"

	err := createTestFile(srcFile, srcFileContent)
	if err != nil {
		t.Errorf("couldn't create test file: %v", err)
	}

	err = createTestFile(dstFile, dstFileContent)
	if err != nil {
		t.Errorf("couldn't create test file: %v", err)
	}

	// cleanup
	defer func() {
		os.Remove(srcFile)
		os.Remove(dstFile)
	}()

	err = FileMove(srcFile, dstFile)
	if err != nil {
		t.Errorf("err is not nil: %v", err)
	}

	_, err = os.Stat(srcFile)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file %s must not exist", srcFile)
	}

	_, err = os.Stat(dstFile)
	if err != nil {
		t.Errorf("file %s must exist", srcFile)
	}

	fileContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Errorf("couldn't reade dstFile content: %v", err)
	}

	if srcFileContent != string(fileContent) {
		t.Errorf("dstFile content is not the same as srcFile content: '%s' != '%s'", fileContent, srcFileContent)
	}
}
