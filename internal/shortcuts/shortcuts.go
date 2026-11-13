// Package shortcuts registers one system-wide hotkey with press and release
// callbacks (release is what push-to-talk needs).
package shortcuts

import (
	"errors"
	"fmt"
	"strings"
)

// Accelerator: lower-case modifiers joined by "+", ending in a KeyboardEvent.code
// name, e.g. "ctrl+option+Space" or "cmd+shift+KeyD". Physical key codes keep
// the shortcut stable across keyboard layouts.
type Accelerator struct {
	Ctrl, Option, Shift, Cmd bool
	Code                     string
}

var ErrUnsupported = errors.New("global shortcuts are not implemented on this platform yet")

func Parse(s string) (Accelerator, error) {
	var a Accelerator
	parts := strings.Split(s, "+")
	for i, p := range parts {
		if i == len(parts)-1 {
			for code := range keyCodes {
				if strings.EqualFold(code, p) { // accept hand-edited or older "space", "keyd"
					a.Code = code
				}
			}
			if a.Code == "" {
				return a, fmt.Errorf("unsupported key %q", p)
			}
			break
		}
		switch strings.ToLower(p) {
		case "ctrl", "control":
			a.Ctrl = true
		case "option", "alt", "opt":
			a.Option = true
		case "shift":
			a.Shift = true
		case "cmd", "command", "meta":
			a.Cmd = true
		default:
			return a, fmt.Errorf("unknown modifier %q", p)
		}
	}
	if !a.Ctrl && !a.Option && !a.Cmd && !strings.HasPrefix(a.Code, "F") {
		return a, errors.New("shortcut needs Ctrl, Option or Cmd (or an F-key) so normal typing isn't hijacked")
	}
	return a, nil
}

// keyCodes maps KeyboardEvent.code to macOS virtual key codes (kVK_*).
var keyCodes = map[string]uint32{
	"KeyA": 0x00, "KeyS": 0x01, "KeyD": 0x02, "KeyF": 0x03, "KeyH": 0x04, "KeyG": 0x05, "KeyZ": 0x06, "KeyX": 0x07,
	"KeyC": 0x08, "KeyV": 0x09, "KeyB": 0x0B, "KeyQ": 0x0C, "KeyW": 0x0D, "KeyE": 0x0E, "KeyR": 0x0F, "KeyY": 0x10,
	"KeyT": 0x11, "Digit1": 0x12, "Digit2": 0x13, "Digit3": 0x14, "Digit4": 0x15, "Digit6": 0x16, "Digit5": 0x17,
	"Equal": 0x18, "Digit9": 0x19, "Digit7": 0x1A, "Minus": 0x1B, "Digit8": 0x1C, "Digit0": 0x1D, "BracketRight": 0x1E,
	"KeyO": 0x1F, "KeyU": 0x20, "BracketLeft": 0x21, "KeyI": 0x22, "KeyP": 0x23, "Enter": 0x24, "KeyL": 0x25,
	"KeyJ": 0x26, "Quote": 0x27, "KeyK": 0x28, "Semicolon": 0x29, "Backslash": 0x2A, "Comma": 0x2B, "Slash": 0x2C,
	"KeyN": 0x2D, "KeyM": 0x2E, "Period": 0x2F, "Tab": 0x30, "Space": 0x31, "Backquote": 0x32, "Escape": 0x35,
	"F1": 0x7A, "F2": 0x78, "F3": 0x63, "F4": 0x76, "F5": 0x60, "F6": 0x61, "F7": 0x62, "F8": 0x64, "F9": 0x65,
	"F10": 0x6D, "F11": 0x67, "F12": 0x6F, "F13": 0x69, "F14": 0x6B, "F15": 0x71, "F16": 0x6A, "F17": 0x40,
	"F18": 0x4F, "F19": 0x50,
}
