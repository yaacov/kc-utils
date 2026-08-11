//go:build linux

package inspect

// LinuxOSReleasePaths are guest-relative paths checked for OS identity.
// usr/lib/os-release is canonical on Fedora and Amazon Linux 2023; /etc/os-release is often a symlink.
var LinuxOSReleasePaths = []string{
	"etc/os-release",
	"usr/lib/os-release",
}

// LinuxRootMarkerPaths are guest-relative paths used to detect a Linux OS root during discovery.
var LinuxRootMarkerPaths = []string{
	"etc/os-release",
	"usr/lib/os-release",
	"etc/redhat-release",
	"etc/debian_version",
}
