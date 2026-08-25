// Package buildinfo exposes compile-time identity shared by AAV binaries.
package buildinfo

const Name = "agent-action-visualizer"

var (
	// Version, Commit, and BuildDate are replaced by release linker flags.
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

func Current() Info {
	return Info{Name: Name, Version: Version, Commit: Commit, BuildDate: BuildDate}
}
