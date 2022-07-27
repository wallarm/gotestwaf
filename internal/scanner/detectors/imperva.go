package detectors

func SecureSphere() *Detector {
	d := &Detector{
		WAFName: "SecureSphere",
		Vendor:  "Imperva Inc.",
	}

	d.Checks = []Check{
		CheckContent("<(title|h2)>Error"),
		CheckContent("The incident ID is"),
		CheckContent("This page can't be displayed"),
		CheckContent("Contact support for additional information"),
	}

	return d
}

func Incapsula() *Detector {
	d := &Detector{
		WAFName: "Incapsula",
		Vendor:  "Imperva Inc.",
	}

	d.Checks = []Check{
		CheckCookie("^incap_ses.*?="),
		CheckCookie("^visid_incap.*?="),
		CheckContent("incapsula incident id"),
		CheckContent("powered by incapsula"),
		CheckContent("/_Incapsula_Resource"),
	}

	return d
}
