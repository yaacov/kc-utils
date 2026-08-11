//go:build linux

package guestagent

// localPackageMajorVersion maps inspect VERSION_ID to an EL major for local
// RPM lookup under rpm/el{N}/. RHEL-family guests use VERSION_ID as the EL
// major; other distros may use unrelated version numbers (e.g. amzn 2023,
// fedora 39).
func localPackageMajorVersion(distro string, major int) int {
	switch distro {
	case "amzn":
		if major >= 2023 {
			return 9
		}
		if major == 2 {
			return 7
		}
		return 9
	case "fedora":
		// VERSION_ID is not an EL major; directory source uses newest EL first.
		return 0
	default:
		return major
	}
}
