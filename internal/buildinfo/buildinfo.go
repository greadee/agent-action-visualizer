// Package buildinfo exposes compile-time identity shared by AAV binaries.
package buildinfo

const (
	// Name is the stable application identifier.
	Name = "agent-action-visualizer"
	// Version is replaced by release builds through linker flags.
	Version = "dev"
)
