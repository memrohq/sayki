//go:build !darwin

// Package install offers to move a downloaded Sayki into /Applications.
package install

// OfferMove is macOS-only.
func OfferMove() (bool, error) { return false, nil }

// EjectInstallers is macOS-only.
func EjectInstallers() {}
