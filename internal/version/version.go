package version

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// GetBuildTime returns the build time
func GetBuildTime() string {
	return BuildTime
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return GitCommit
}