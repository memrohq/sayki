//go:build !darwin

package shortcuts

// shortcut: Windows (RegisterHotKey + low-level hook for key-up) and Linux (X11
// XGrabKey; Wayland only via the GlobalShortcuts portal) are not implemented yet.
func Register(Accelerator, func(pressed bool)) error { return ErrUnsupported }
func Unregister()                                    {}
