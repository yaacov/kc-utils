//go:build unix

package guestcleanup

// Run removes stale guest caches and modprobe aliases after conversion.
func Run(guestRoot string) {
	Clean(guestRoot)
	Configure(guestRoot)
}
