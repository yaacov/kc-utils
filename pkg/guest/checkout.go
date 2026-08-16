//go:build unix

package guest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// liveHostTree is implemented by backends that expose a real host mount tree
// (direct). QEMU and guestfs use download/upload instead.
type liveHostTree interface {
	LiveHostPath(guestPath string) (string, bool)
}

func (g *Guest) liveHostPath(guestPath string) (string, bool) {
	if t, ok := g.backend.(liveHostTree); ok {
		return t.LiveHostPath(guestPath)
	}
	return "", false
}

// MergeHive applies a .reg snippet to a guest hive via the active backend.
func (g *Guest) MergeHive(guestPath string, reg []byte) error {
	if len(bytes.TrimSpace(reg)) == 0 {
		return nil
	}
	return g.backend.MergeHive(normalizeGuestPath(guestPath), reg)
}

// Checkout downloads a guest file to a host temp path for tools that need a
// real filesystem path (e.g. hivex). Live-host-tree backends return HostPath.
// Caller must Checkin or DiscardCheckout when done.
func (g *Guest) Checkout(guestPath string) (hostPath string, err error) {
	guestPath = normalizeGuestPath(guestPath)
	if p, ok := g.liveHostPath(guestPath); ok {
		return p, nil
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
// the temp file. No-op on live-host-tree backends (edits were live).
func (g *Guest) Checkin(guestPath, hostPath string) error {
	guestPath = normalizeGuestPath(guestPath)
	if _, ok := g.liveHostPath(guestPath); ok {
		return nil
	}
	if err := g.backend.Upload(hostPath, guestPath); err != nil {
		return fmt.Errorf("checkin %s: %w", guestPath, err)
	}
	_ = os.Remove(hostPath)
	return nil
}

// CheckoutReadOnly downloads a guest file to a host temp path for read-only
// tools. The returned cleanup function removes the temp file or is a no-op
// on a live host tree. Caller must call cleanup when done; no Checkin is needed.
func (g *Guest) CheckoutReadOnly(guestPath string) (hostPath string, cleanup func(), err error) {
	guestPath = normalizeGuestPath(guestPath)
	if p, ok := g.liveHostPath(guestPath); ok {
		return p, func() {}, nil
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

// DiscardCheckout removes a checkout temp without uploading. No-op on a live
// host tree when hostPath is the live mount path.
func (g *Guest) DiscardCheckout(hostPath string) {
	if _, ok := g.liveHostPath("/"); ok {
		return
	}
	_ = os.Remove(hostPath)
}
