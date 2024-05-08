package detectors

func KonaSiteDefender() *Detector {
	d := &Detector{
		WAFName: "Kona SiteDefender",
		Vendor:  "Akamai",
	}

	d.Check = Or(
		CheckHeader("Server", "AkamaiGHost", false),
		CheckHeader("Server", "AkamaiGHost", true),
	)

	return d
}
