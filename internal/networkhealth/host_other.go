//go:build !linux

package networkhealth

// Snapshot does not perform active host inspection on unsupported platforms.
func Snapshot() Host {
	return Host{Platform: "unsupported", Status: "unsupported", Codes: []string{"LINUX_REQUIRED"}}
}
