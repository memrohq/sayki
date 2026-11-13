// Package sound plays short feedback cues using macOS's built-in system sounds.
package sound

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#include <stdlib.h>

static void playNamed(const char *name, float volume) {
	NSSound *s = [[NSSound soundNamed:[NSString stringWithUTF8String:name]] copy]; // copy: overlapping cues can't cut each other off
	s.volume = volume;
	[s play]; // asynchronous
}
*/
import "C"

import "unsafe"

// Cues, named after files in /System/Library/Sounds.
const (
	Start  = "Tink"
	Stop   = "Pop"
	Cancel = "Bottle"
)

// Play starts a cue and returns immediately.
func Play(name string) {
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	C.playNamed(c, 0.6)
}
