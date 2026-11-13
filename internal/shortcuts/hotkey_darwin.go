package shortcuts

/*
#cgo LDFLAGS: -framework Carbon -framework Foundation
#include <Carbon/Carbon.h>
#include <dispatch/dispatch.h>
#include <pthread.h>

extern void goHotkey(int pressed);
extern void goFnKey(int down);
extern void goFnOther(void);

static EventHotKeyRef hkRef = NULL;
static EventHandlerRef handlerRef = NULL;

static OSStatus onHotkey(EventHandlerCallRef next, EventRef ev, void *data) {
	goHotkey(GetEventKind(ev) == kEventHotKeyPressed);
	return noErr;
}

static void onMain(dispatch_block_t b) {
	if (pthread_main_np()) b(); else dispatch_sync(dispatch_get_main_queue(), b);
}

static int registerHotkey(UInt32 code, UInt32 mods) {
	__block OSStatus st = noErr;
	onMain(^{
		if (!handlerRef) {
			EventTypeSpec types[2] = {{kEventClassKeyboard, kEventHotKeyPressed}, {kEventClassKeyboard, kEventHotKeyReleased}};
			InstallApplicationEventHandler(&onHotkey, 2, types, NULL, &handlerRef);
		}
		if (hkRef) { UnregisterEventHotKey(hkRef); hkRef = NULL; }
		EventHotKeyID id = {'SAYK', 1};
		st = RegisterEventHotKey(code, mods, id, GetApplicationEventTarget(), 0, &hkRef);
	});
	return (int)st;
}

static void unregisterHotkey(void) {
	onMain(^{ if (hkRef) { UnregisterEventHotKey(hkRef); hkRef = NULL; } });
}

// Single keys (Fn, Control, right Option/Command) can't be Carbon hotkeys, so a
// listen-only event tap watches them. Needs Accessibility, which Sayki already
// requires to type.
static CFMachPortRef fnTap = NULL;
static CFRunLoopSourceRef fnSrc = NULL;
static int64_t soloA = -1, soloB = -1; // key codes that count as the solo key
static CGEventFlags soloFlag = 0;      // modifier flag that is set while it's down

static CGEventRef onFnEvent(CGEventTapProxy proxy, CGEventType type, CGEventRef ev, void *ctx) {
	if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
		if (fnTap) CGEventTapEnable(fnTap, true);
		return ev;
	}
	int64_t code = CGEventGetIntegerValueField(ev, kCGKeyboardEventKeycode);
	if (type == kCGEventFlagsChanged && (code == soloA || code == soloB)) {
		goFnKey((CGEventGetFlags(ev) & soloFlag) != 0);
	} else {
		goFnOther(); // any other key or modifier: the solo key is being used as a modifier
	}
	return ev;
}

static int startFnTap(int64_t a, int64_t b, CGEventFlags flag) {
	soloA = a; soloB = b; soloFlag = flag;
	__block int ok = 1;
	onMain(^{
		if (fnTap) return;
		CGEventMask mask = CGEventMaskBit(kCGEventFlagsChanged) | CGEventMaskBit(kCGEventKeyDown);
		fnTap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionListenOnly, mask, onFnEvent, NULL);
		if (!fnTap) { ok = 0; return; }
		fnSrc = CFMachPortCreateRunLoopSource(NULL, fnTap, 0);
		CFRunLoopAddSource(CFRunLoopGetMain(), fnSrc, kCFRunLoopCommonModes);
		CGEventTapEnable(fnTap, true);
	});
	return ok;
}

static void stopFnTap(void) {
	onMain(^{
		if (!fnTap) return;
		CGEventTapEnable(fnTap, false);
		CFRunLoopRemoveSource(CFRunLoopGetMain(), fnSrc, kCFRunLoopCommonModes);
		CFRelease(fnSrc); CFRelease(fnTap);
		fnSrc = NULL; fnTap = NULL;
	});
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
)

var (
	mu      sync.Mutex
	handler func(pressed bool)
	gate    *fnGate // non-nil while a solo key is the shortcut
)

// soloCodes maps SoloKeys to macOS key codes (kVK_*) and the flag set while held.
var soloCodes = map[string]struct {
	a, b C.int64_t
	flag C.CGEventFlags
}{
	"fn":          {C.kVK_Function, -1, C.kCGEventFlagMaskSecondaryFn},
	"ctrl":        {C.kVK_Control, C.kVK_RightControl, C.kCGEventFlagMaskControl},
	"rightoption": {C.kVK_RightOption, -1, C.kCGEventFlagMaskAlternate},
	"rightcmd":    {C.kVK_RightCommand, -1, C.kCGEventFlagMaskCommand},
}

//export goFnKey
func goFnKey(down C.int) {
	mu.Lock()
	g := gate
	mu.Unlock()
	if g != nil {
		g.key(down != 0)
	}
}

//export goFnOther
func goFnOther() {
	mu.Lock()
	g := gate
	mu.Unlock()
	if g != nil {
		g.other()
	}
}

//export goHotkey
func goHotkey(pressed C.int) {
	mu.Lock()
	h := handler
	mu.Unlock()
	if h != nil {
		go h(pressed != 0) // never block the main run loop
	}
}

// Register replaces the current global hotkey. fn runs on its own goroutine.
func Register(acc Accelerator, fn func(pressed bool)) error {
	Unregister()
	if acc.Solo != "" {
		k, ok := soloCodes[acc.Solo]
		if !ok {
			return fmt.Errorf("unsupported key %q", acc.Solo)
		}
		mu.Lock()
		gate = newFnGate(fnHold, fn)
		mu.Unlock()
		if C.startFnTap(k.a, k.b, k.flag) == 0 {
			Unregister()
			return errors.New("single-key shortcuts need Accessibility access: enable Sayki in System Settings → Privacy & Security → Accessibility")
		}
		return nil
	}
	var mods C.UInt32
	if acc.Cmd {
		mods |= C.cmdKey
	}
	if acc.Shift {
		mods |= C.shiftKey
	}
	if acc.Option {
		mods |= C.optionKey
	}
	if acc.Ctrl {
		mods |= C.controlKey
	}
	mu.Lock()
	handler = fn
	mu.Unlock()
	if st := C.registerHotkey(C.UInt32(keyCodes[acc.Code]), mods); st != 0 {
		return fmt.Errorf("shortcut is already taken by the system or another app (OSStatus %d)", int(st))
	}
	return nil
}

func Unregister() {
	C.unregisterHotkey()
	C.stopFnTap()
	mu.Lock()
	g := gate
	gate = nil
	mu.Unlock()
	if g != nil {
		g.close()
	}
}
