//go:build linux

package guest

import (
	"fmt"
	"os"
	"path/filepath"
)

// Checkout downloads a guest file to a host temp path for tools that need a
// real filesystem path (e.g. hivex). In direct mode returns HostPath (live mount).
// Caller must Checkin or DiscardCheckout when done.
func (g *Guest) Checkout(guestPath string) (hostPath string, err error) {
	guestPath = normalizeGuestPath(guestPath)
	if g.mode == ModeDirect {
		return g.HostPath(guestPath), nil
	}
	base := filepath.Base(guestPath)
	tmp, err := os.CreateTemp("", "kc-checkout-"+base+"-*")
	if err != nil {
		return "", fmt.Errorf("checkout %s: %w", guestPath, err)
	}
	hostPath = tmp.Name()
	tmp.Close()
	if err := g.backend.Download(guestPath, hostPath); err != nil {
		os.Remove(hostPath)
		return "", fmt.Errorf("checkout %s: %w", guestPath, err)
	}
	return hostPath, nil
}

// Checkin uploads a checked-out host file back to the guest path and removes
// the temp file. In direct mode this is a no-op (edits were live).
func (g *Guest) Checkin(guestPath, hostPath string) error {
	guestPath = normalizeGuestPath(guestPath)
	if g.mode == ModeDirect {
		return nil
	}
	if err := g.backend.Upload(hostPath, guestPath); err != nil {
		return fmt.Errorf("checkin %s: %w", guestPath, err)
	}
	_ = os.Remove(hostPath)
	return nil
}

// CheckoutReadOnly downloads a guest file to a host temp path for read-only
// tools. The returned cleanup function removes the temp file (guestfs) or is a
// no-op (direct). Caller must call cleanup when done; no Checkin is needed.
func (g *Guest) CheckoutReadOnly(guestPath string) (hostPath string, cleanup func(), err error) {
	guestPath = normalizeGuestPath(guestPath)
	if g.mode == ModeDirect {
		return g.HostPath(guestPath), func() {}, nil
	}
	base := filepath.Base(guestPath)
	tmp, err := os.CreateTemp("", "kc-checkout-ro-"+base+"-*")
	if err != nil {
		return "", nil, fmt.Errorf("checkout-ro %s: %w", guestPath, err)
	}
	hostPath = tmp.Name()
	tmp.Close()
	if err := g.backend.Download(guestPath, hostPath); err != nil {
		os.Remove(hostPath)
		return "", nil, fmt.Errorf("checkout-ro %s: %w", guestPath, err)
	}
	return hostPath, func() { os.Remove(hostPath) }, nil
}

// DiscardCheckout removes a checkout temp without uploading. No-op in direct mode
// when hostPath is the live mount path.
func (g *Guest) DiscardCheckout(hostPath string) {
	if g.mode == ModeDirect {
		return
	}
	_ = os.Remove(hostPath)
}
