package injection

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework Carbon
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>

static int trusted(void) { return AXIsProcessTrusted(); }
static int secureInput(void) { return IsSecureEventInputEnabled(); }

static CGPoint cursor(void) {
	CGEventRef e = CGEventCreate(NULL);
	CGPoint p = CGEventGetLocation(e); // global, top-left origin, points
	CFRelease(e);
	return p;
}

static int modifiersHeld(void) {
	CGEventFlags f = CGEventSourceFlagsState(kCGEventSourceStateHIDSystemState);
	return (f & (kCGEventFlagMaskCommand|kCGEventFlagMaskControl|kCGEventFlagMaskAlternate|kCGEventFlagMaskShift|kCGEventFlagMaskSecondaryFn)) != 0;
}

static void onMain(dispatch_block_t b) {
	if ([NSThread isMainThread]) b(); else dispatch_sync(dispatch_get_main_queue(), b);
}

static pid_t frontmostPID(void) {
	__block pid_t pid = 0;
	onMain(^{ pid = NSWorkspace.sharedWorkspace.frontmostApplication.processIdentifier; });
	return pid;
}

static void activatePID(pid_t pid) {
	onMain(^{ [[NSRunningApplication runningApplicationWithProcessIdentifier:pid] activateWithOptions:0]; });
}

static void postKey(CGKeyCode code, CGEventFlags flags) {
	CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStatePrivate);
	CGEventRef down = CGEventCreateKeyboardEvent(src, code, true);
	CGEventRef up = CGEventCreateKeyboardEvent(src, code, false);
	CGEventSetFlags(down, flags);
	CGEventSetFlags(up, flags);
	CGEventPost(kCGHIDEventTap, down);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down); CFRelease(up);
	if (src) CFRelease(src);
}

// typeUTF16 posts the text as unicode keystrokes, 20 UTF-16 units per event (the API limit).
static void typeUTF16(const UniChar *s, int n) {
	CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStatePrivate);
	for (int i = 0, len; i < n; i += len) {
		len = n - i < 20 ? n - i : 20;
		if (len == 20 && CFStringIsSurrogateHighCharacter(s[i + 19])) len = 19; // don't split a pair
		CGEventRef down = CGEventCreateKeyboardEvent(src, 0, true);
		CGEventRef up = CGEventCreateKeyboardEvent(src, 0, false);
		CGEventSetFlags(down, 0); CGEventSetFlags(up, 0);
		CGEventKeyboardSetUnicodeString(down, len, s + i);
		CGEventKeyboardSetUnicodeString(up, len, s + i);
		CGEventPost(kCGHIDEventTap, down);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(down); CFRelease(up);
		usleep(2000);  // some terminals drop events that arrive in one burst
	}
	if (src) CFRelease(src);
}

static NSArray<NSPasteboardItem*> *saved;

// pasteText snapshots every pasteboard item/type, puts text on it and sends Cmd+V.
// Returns the changeCount after our write, to detect if anyone else touched it.
static long pasteText(const char *utf8) {
	__block long cc = 0;
	onMain(^{
		NSPasteboard *pb = NSPasteboard.generalPasteboard;
		NSMutableArray *copy = [NSMutableArray array];
		for (NSPasteboardItem *item in pb.pasteboardItems) {
			NSPasteboardItem *c = [NSPasteboardItem new];
			for (NSPasteboardType t in item.types) {
				NSData *d = [item dataForType:t];
				if (d) [c setData:d forType:t];
			}
			[copy addObject:c];
		}
		saved = copy;
		[pb clearContents];
		NSPasteboardItem *it = [NSPasteboardItem new];
		[it setString:[NSString stringWithUTF8String:utf8] forType:NSPasteboardTypeString];
		// Ask clipboard managers not to record dictated text (nspasteboard.org convention).
		[it setString:@"" forType:@"org.nspasteboard.TransientType"];
		[pb writeObjects:@[it]];
		cc = pb.changeCount;
	});
	postKey(kVK_ANSI_V, kCGEventFlagMaskCommand);
	return cc;
}

static void restorePasteboard(long expected) {
	onMain(^{
		NSPasteboard *pb = NSPasteboard.generalPasteboard;
		if (pb.changeCount == expected) { // user/app copied something new meanwhile: keep theirs
			[pb clearContents];
			if (saved.count) [pb writeObjects:saved];
		}
		saved = nil;
	});
}
*/
import "C"

import (
	"os"
	"time"
	"unicode/utf16"
	"unsafe"
)

// Target remembers which app had focus when dictation started.
type Target struct{ pid C.pid_t }

func CaptureTarget() Target { return Target{pid: C.frontmostPID()} }

func Trusted() bool { return C.trusted() != 0 }

// Cursor returns the mouse position in global screen points (top-left origin).
func Cursor() (x, y int) {
	p := C.cursor()
	return int(p.x), int(p.y)
}

// Insert types text into the focused app. method is "paste" or "type".
func Insert(t Target, text, method string) error {
	text = Sanitize(text)
	if text == "" {
		return nil
	}
	if !Trusted() {
		return ErrNoAccessibility
	}
	// Push-to-talk users may still be holding the shortcut's modifiers; a
	// synthetic Cmd+V combined with held Ctrl/Option would be a different shortcut.
	for deadline := time.Now().Add(1500 * time.Millisecond); C.modifiersHeld() != 0 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if t.pid != 0 && C.frontmostPID() != t.pid && C.frontmostPID() == C.pid_t(os.Getpid()) {
		// Our own window took focus (e.g. tray menu); hand it back before typing.
		C.activatePID(t.pid)
		time.Sleep(150 * time.Millisecond)
	}
	if C.secureInput() != 0 {
		return ErrSecureInput
	}
	if method == "type" {
		u := utf16.Encode([]rune(text))
		C.typeUTF16((*C.UniChar)(unsafe.Pointer(&u[0])), C.int(len(u)))
		return nil
	}
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	cc := C.pasteText(cs)
	// The target reads the pasteboard asynchronously after Cmd+V; give it time.
	time.Sleep(350 * time.Millisecond)
	C.restorePasteboard(cc)
	return nil
}
