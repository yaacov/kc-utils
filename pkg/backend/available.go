//go:build unix

package backend

// Available reports whether the plugin's runtime requirements are satisfied.
func Available(p Plugin) bool {
	ok, _ := checkRequirements(p.Requirements())
	return ok
}
