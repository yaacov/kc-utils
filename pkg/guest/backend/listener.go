//go:build unix

package backend

// Environment variables for the guestfs shared listener and Clevis networking.
// Defined here so callers need not import a concrete backend package.
const (
	EnvGuestfishPID   = "GUESTFISH_PID"
	EnvKCGuestfishPID = "KC_GUESTFISH_PID"
	// EnvGuestfsNetwork enables QEMU user networking in the appliance before
	// launch. Set to "1" or "true" when Clevis/NBDE unlock is required.
	EnvGuestfsNetwork = "KC_GUESTFS_NETWORK"
	// EnvAgentSock is the Unix socket path for the qemu backend kc-agent.
	EnvAgentSock = "KC_AGENT_SOCK"
	// EnvQemuPID is the QEMU process id for the qemu shared session.
	EnvQemuPID = "KC_QEMU_PID"
	// EnvApplianceDir is the directory containing vmlinuz and initramfs.img.
	EnvApplianceDir = "KC_APPLIANCE_DIR"
	// EnvVirtioWin overrides the host virtio-win tree (same for all backends).
	EnvVirtioWin = "KC_VIRTIO_WIN"
	// EnvKCPackages overrides the host qemu-ga package tree.
	EnvKCPackages = "KC_PACKAGES"
)

// StartSharedSession starts a shared backend session when the selected
// backend implements SharedSessionFactory.
func StartSharedSession(backend string) (SharedSession, error) {
	f, err := LookupFactory(backend)
	if err != nil {
		return nil, err
	}
	sf, ok := f.(SharedSessionFactory)
	if !ok {
		return nil, nil
	}
	return sf.StartSharedSession()
}

// StartSharedListener starts a guestfs shared guestfish --listen session.
// Orchestrators that already know the configured backend should call
// StartSharedSession(backend) instead so restarts keep the same backend.
func StartSharedListener() (SharedSession, error) {
	return StartSharedSession(string(ModeGuestfs))
}

// SharedListenerAlive reports whether the session is still running.
func SharedListenerAlive(l SharedSession) bool {
	return l != nil && l.Alive()
}
