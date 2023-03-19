package detectors

func Cloudflare() *Detector {
	d := &Detector{
		WAFName: "Cloudflare",
		Vendor:  "Cloudflare Inc",
	}

	d.Checks = []Check{
		CheckStatusCode(403),
		CheckHeader("Server", "cloudflare"),
		CheckContent("Cloudflare Ray ID"),
	}

	return d
}
