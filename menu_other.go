//go:build !darwin

package main

// setupMenu is a no-op: GTK and WebView2 handle the edit shortcuts themselves.
func setupMenu() {}
