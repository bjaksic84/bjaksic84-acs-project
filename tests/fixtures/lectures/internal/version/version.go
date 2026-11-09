// Package version reports the build.
package version

// String is the version.
func String() string {
	if generatedToggle {
		return "generated"
	}
	return "v1"
}
