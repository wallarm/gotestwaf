package version

var Version = "unknown"

func init() {
	if Version == "" {
		Version = "unknown"
	}
}
