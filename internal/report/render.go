package report

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/pkg/errors"
)

// chromium-browser \
// --headless \
// --no-zygote \
// --single-process \
// --no-sandbox \
// --disable-gpu \
// --run-all-compositor-stages-before-draw \
// --print-to-pdf-no-header \
// --print-to-pdf=test.pdf \
// report.html

func findChrome() (string, error) {
	var chromePath string
	switch runtime.GOOS {
	case "windows":
		chromePath = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
	case "darwin":
		chromePath = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	case "linux":
		var err error
		for _, name := range []string{"chromium-browser", "google-chrome-stable"} {
			chromePath, err = exec.LookPath(name)
			if err == nil {
				break
			}
		}
	}

	if chromePath == "" {
		return "", errors.New("chrome not found")
	}

	if _, err := os.Stat(chromePath); errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return chromePath, nil
}

func renderToPDF(ctx context.Context, fileToRender string, pathToResultPDF string) error {
	chromePath, err := findChrome()
	if err != nil {
		return errors.Wrap(err, "couldn't find Chrome/Chromium to render HTML file to PDF")
	}

	cmd := exec.CommandContext(ctx, chromePath,
		"--headless",
		"--no-zygote",
		"--single-process",
		"--no-sandbox",
		"--disable-gpu",
		"--run-all-compositor-stages-before-draw",
		"--print-to-pdf-no-header",
		fmt.Sprintf("--print-to-pdf=%s", pathToResultPDF),
		fileToRender,
	)

	err = cmd.Run()

	return err
}
