package detectors

func BigIPAppSecManager() *Detector {
	d := &Detector{
		WAFName: "BIG-IP AppSec Manager",
		Vendor:  "F5 Networks",
	}

	d.Check = Or(
		And(
			CheckContent("the requested url was rejected", true),
			CheckContent("please consult with your administrator", true),
		),
		CheckCookie("^TS.+?", false),
		CheckContent("Reference ID", true),
	)

	return d
}

func BigIPLocalTrafficManager() *Detector {
	d := &Detector{
		WAFName: "BIG-IP Local Traffic Manager",
		Vendor:  "F5 Networks",
	}

	d.Check = Or(
		CheckCookie("^bigipserver", false),
		CheckHeader("X-Cnection", "close", true),
	)

	return d
}

func BigIPApManager() *Detector {
	d := &Detector{
		WAFName: "BIG-IP AP Manager",
		Vendor:  "F5 Networks",
	}

	d.Check = Or(
		And(
			CheckCookie("^LastMRH_Session", false),
			CheckCookie("^MRHSession", false),
		),
		And(
			CheckCookie("^MRHSession", false),
			CheckHeader("Server", "Big([-_])?IP", true),
		),
		Or(
			CheckCookie("^F5_fullWT", false),
			CheckCookie("^F5_HT_shrinked", false),
		),
	)

	return d
}

func FirePass() *Detector {
	d := &Detector{
		WAFName: "FirePass",
		Vendor:  "F5 Networks",
	}

	d.Check = Or(
		And(
			CheckCookie("^VHOST", false),
			CheckHeader("Location", `\/my\.logon\.php3`, false),
		),
		And(
			CheckCookie("^F5_fire.+?", false),
			CheckCookie("^F5_passid_shrinked", false),
		),
	)

	return d
}

func Trafficshield() *Detector {
	d := &Detector{
		WAFName: "Trafficshield",
		Vendor:  "F5 Networks",
	}

	d.Check = Or(
		CheckCookie("^ASINFO=", false),
		CheckHeader("Server", "F5-TrafficShield", false),
	)

	return d
}
