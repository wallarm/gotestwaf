package detectors

func KonaSiteDefender() *Detector {
	d := &Detector{
		WAFName: "Kona SiteDefender",
		Vendor:  "Akamai",
	}

	d.Checks = []Check{
		CheckHeader("Server", "AkamaiGHost"),
	}

	return d
}
