package platform

import "github.com/schollz/progressbar/v3"

func NewProgressBar() *progressbar.ProgressBar {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("WAF Scanning"),
		progressbar.OptionSetItsString("test"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionClearOnFinish())
	return bar
}
