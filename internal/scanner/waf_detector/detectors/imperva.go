package detectors

func SecureSphere() *Detector {
	d := &Detector{
		WAFName: "SecureSphere",
		Vendor:  "Imperva Inc.",
	}

	d.Check = And(
		CheckContent("<(title|h2)>Error", true),
		CheckContent("The incident ID is", true),
		CheckContent("This page can't be displayed", true),
		CheckContent("Contact support for additional information", true),
	)

	return d
}

func Incapsula() *Detector {
	d := &Detector{
		WAFName: "Incapsula",
		Vendor:  "Imperva Inc.",
	}

	d.Check = Or(
		CheckCookie("^incap_ses.*?=", false),
		CheckCookie("^visid_incap.*?=", false),
		CheckContent("incapsula incident id", true),
		CheckContent("powered by incapsula", true),
		CheckContent("/_Incapsula_Resource", true),
	)

	return d
}
