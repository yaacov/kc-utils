package version

// handlersWithoutGuestAgent lists version handlers that must not collect or
// install qemu-ga MSIs (virt-v2v parity: XP, 2003, Server 2008, Vista).
var handlersWithoutGuestAgent = map[string]struct{}{
	"win2008":  {},
	"winvista": {},
	"win2003":  {},
	"winxp":    {},
}

// CollectGuestAgentMSI reports whether qemu-ga MSIs should be included in
// CollectDrivers results for the given handler name.
func CollectGuestAgentMSI(handlerName string) bool {
	if handlerName == "" {
		return true
	}
	_, skip := handlersWithoutGuestAgent[handlerName]
	return !skip
}
