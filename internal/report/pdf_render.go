package report

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
)

// chromium-browser
// --headless
// --disable-gpu
// --run-all-compositor-stages-before-draw
// --no-sandbox
// --print-to-pdf-no-header
// --print-to-pdf=test.pdf
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

func renderToPDF(html []byte, pathToResultPDF string) error {
	chromePath, err := findChrome()
	if err != nil {
		return err
	}

	file, err := ioutil.TempFile("", "gotestwaf_report_*.html")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	file.Write(html)
	file.Close()

	cmd := exec.Command(chromePath,
		"--headless",
		"--disable-gpu",
		"--run-all-compositor-stages-before-draw",
		"--no-sandbox",
		"--print-to-pdf-no-header",
		fmt.Sprintf("--print-to-pdf=%s", pathToResultPDF),
		file.Name(),
	)

	err = cmd.Run()

	return err
}
