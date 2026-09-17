//go:build !linux

package platform

// ConfigureRendering is a no-op outside Linux.
func ConfigureRendering() {}
