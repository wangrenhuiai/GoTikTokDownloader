//go:build !windows

package browser

// RuntimeVersion stub for non-Windows builds.
func RuntimeVersion() string { return "unsupported platform" }
