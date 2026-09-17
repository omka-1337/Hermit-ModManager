//go:build !linux || !cgo || gtk3

package platform

import "unsafe"

// PrepareFramelessWindow is only needed with GTK 4 on Linux.
func PrepareFramelessWindow(unsafe.Pointer, string) {}

// SetProgramName is only needed with GTK on Linux.
func SetProgramName(string, string) {}
