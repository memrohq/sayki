//go:build !darwin

// Package sound plays short feedback cues using macOS's built-in system sounds.
package sound

const (
	Start  = "Tink"
	Stop   = "Pop"
	Cancel = "Bottle"
)

// shortcut: no sound on other platforms yet.
func Play(string) {}
