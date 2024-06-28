package detectors

// Detector contains names of WAF solution and vendor, and checks to detect that
// solution by response.
type Detector struct {
	WAFName string
	Vendor  string

	Check Check
}

func (d *Detector) GetWAFName() string {
	return d.WAFName
}

func (d *Detector) GetVendor() string {
	return d.Vendor
}

func (d *Detector) IsWAF(resps *Responses) bool {
	return d.Check(resps)
}

// Detectors is the list of all available WAF detectors. The checks are performed
// in the given order.
var Detectors = []*Detector{
	// Akamai
	KonaSiteDefender(),

	// Imperva
	Incapsula(),
	SecureSphere(),

	// F5 Networks
	BigIPAppSecManager(),
	BigIPLocalTrafficManager(),
	BigIPApManager(),
	FirePass(),
	Trafficshield(),

	ModSecurity(),
}
