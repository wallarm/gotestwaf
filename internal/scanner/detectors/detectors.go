package detectors

import "net/http"

// Detector contains names of WAF solution and vendor, and checks to detect that
// solution by response.
type Detector struct {
	WAFName string
	Vendor  string

	Checks []Check
}

func (d *Detector) GetWAFName() string {
	return d.WAFName
}

func (d *Detector) GetVendor() string {
	return d.Vendor
}

func (d *Detector) IsWAF(resp *http.Response) bool {
	for _, check := range d.Checks {
		if check(resp) {
			return true
		}
	}

	return false
}

// Detectors is the list of all available WAF detectors. The checks are performed
// in the given order.
var Detectors = []*Detector{
	// Akamai
	KonaSiteDefender(),

	// Imperva
	Incapsula(),
	SecureSphere(),
}
