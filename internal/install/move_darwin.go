// Package install offers to move a downloaded Sayki into /Applications, so a
// ZIP download needs no disk image: unzip, open, click Move.
package install

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Security
#import <AppKit/AppKit.h>
#include <sys/xattr.h>
#include <stdlib.h>

// macOS runs quarantined apps from a hidden read-only copy ("App Translocation").
// These Security.framework calls map it back to the file the user opened.
extern Boolean SecTranslocateIsTranslocatedURL(CFURLRef path, bool *isTranslocated, CFErrorRef *error) __attribute__((weak_import));
extern CFURLRef SecTranslocateCreateOriginalPathForURL(CFURLRef translocatedPath, CFErrorRef *error) __attribute__((weak_import));

static NSString *originalBundlePath(void) {
	NSURL *u = NSBundle.mainBundle.bundleURL;
	bool translocated = false;
	if (SecTranslocateIsTranslocatedURL && SecTranslocateIsTranslocatedURL((__bridge CFURLRef)u, &translocated, NULL) && translocated) {
		CFURLRef orig = SecTranslocateCreateOriginalPathForURL((__bridge CFURLRef)u, NULL);
		if (orig) return ((__bridge_transfer NSURL *)orig).path;
	}
	return u.path;
}

// 0: nothing to do or declined; 1: moved and relaunching, quit now; -1: failed (*err set).
static int offerMove(char **err) {
	NSString *src = originalBundlePath();
	NSString *dst = @"/Applications/Sayki.app";
	if ([src hasPrefix:@"/Applications/"] || [src hasPrefix:[NSHomeDirectory() stringByAppendingString:@"/Applications/"]]) return 0;
	// Only downloaded copies carry a quarantine flag, so development builds never prompt.
	if (getxattr(src.fileSystemRepresentation, "com.apple.quarantine", NULL, 0, 0, 0) < 0) return 0;

	NSAlert *a = [NSAlert new];
	a.messageText = @"Move Sayki to your Applications folder?";
	a.informativeText = @"Sayki works best from Applications, and macOS keeps its permissions there. The downloaded copy will be moved to the Trash.";
	[a addButtonWithTitle:@"Move to Applications"];
	[a addButtonWithTitle:@"Not Now"];
	[NSApp activateIgnoringOtherApps:YES];
	if ([a runModal] != NSAlertFirstButtonReturn) return 0;

	NSFileManager *fm = NSFileManager.defaultManager;
	NSError *e = nil;
	if ([fm fileExistsAtPath:dst] && ![fm trashItemAtURL:[NSURL fileURLWithPath:dst] resultingItemURL:nil error:&e]) {
		*err = strdup(e.localizedDescription.UTF8String);
		return -1;
	}
	if (![fm copyItemAtPath:src toPath:dst error:&e]) {
		*err = strdup(e.localizedDescription.UTF8String);
		return -1;
	}
	[fm trashItemAtURL:[NSURL fileURLWithPath:src] resultingItemURL:nil error:nil]; // best effort: it may be read-only

	// Relaunch from Applications once this copy has quit (Sayki allows one copy at a time).
	NSTask *t = [NSTask new];
	t.executableURL = [NSURL fileURLWithPath:@"/bin/sh"];
	t.arguments = @[@"-c", @"sleep 1; /usr/bin/open \"$0\"", dst];
	[t launchAndReturnError:nil];
	return 1;
}

static void showError(const char *msg) {
	NSAlert *a = [NSAlert new];
	a.messageText = @"Couldn't move Sayki";
	a.informativeText = [NSString stringWithFormat:@"%s\n\nDrag Sayki into your Applications folder yourself instead.", msg];
	[a runModal];
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// OfferMove asks to move a downloaded copy of Sayki into /Applications. It
// returns true when the app was moved and is relaunching, so the caller must
// quit. It must run on the main thread.
func OfferMove() (bool, error) {
	var cerr *C.char
	switch C.offerMove(&cerr) {
	case 1:
		return true, nil
	case -1:
		msg := C.GoString(cerr)
		C.showError(cerr)
		C.free(unsafe.Pointer(cerr))
		return false, errors.New(msg)
	}
	return false, nil
}
