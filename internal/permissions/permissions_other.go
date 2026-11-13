//go:build !darwin

package permissions

import "errors"

// shortcut: Windows mic privacy (ms-settings:privacy-microphone) and Linux have no checks yet.
func Check() Status             { return Status{Microphone: "unsupported", Accessibility: "unsupported"} }
func RequestMicrophone()        {}
func RequestAccessibility()     {}
func OpenSettings(string) error { return errors.New("not supported on this platform") }
