//go:build !darwin

package injection

// shortcut: Windows (SendInput with KEYEVENTF_UNICODE) and Linux (X11 XTest;
// Wayland needs the RemoteDesktop portal or ydotool) are not implemented yet.
type Target struct{}

func CaptureTarget() Target               { return Target{} }
func Trusted() bool                       { return false }
func Cursor() (x, y int)                  { return 0, 0 }
func Insert(Target, string, string) error { return ErrUnsupported }
