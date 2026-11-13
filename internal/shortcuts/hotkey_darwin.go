package shortcuts

/*
#cgo LDFLAGS: -framework Carbon -framework Foundation
#include <Carbon/Carbon.h>
#include <dispatch/dispatch.h>
#include <pthread.h>

extern void goHotkey(int pressed);

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
*/
import "C"

import (
	"fmt"
	"sync"
)

var (
	mu      sync.Mutex
	handler func(pressed bool)
)

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

func Unregister() { C.unregisterHotkey() }
