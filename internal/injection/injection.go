// Package injection inserts text at the cursor of whatever app currently has
// keyboard focus. It never sends Return/Enter, so terminal commands are not run.
package injection

import (
	"errors"
	"strings"
)

var (
	ErrUnsupported     = errors.New("text insertion is not implemented on this platform yet")
	ErrNoAccessibility = errors.New("Sayki needs Accessibility permission to type into other apps")
	ErrSecureInput     = errors.New("the focused field is a secure (password) field; Sayki won't type there")
)

// Sanitize strips control characters (including newlines) so inserted text can
// never submit a terminal command or trigger shortcuts.
func Sanitize(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, text)
}
