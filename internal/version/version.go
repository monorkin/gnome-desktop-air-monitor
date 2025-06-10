package version

// Build-time variable (set via ldflags)
var Version = "dev"

// GetVersion returns the current version
func GetVersion() string {
	return Version
}