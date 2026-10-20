//go:build !darwin && !linux

package main

// pickFiles is not implemented on this platform.
func pickFiles(dir string) []string { return nil }
