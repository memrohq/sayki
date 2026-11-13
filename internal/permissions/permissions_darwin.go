package permissions

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework ApplicationServices
#import <AVFoundation/AVFoundation.h>
#import <ApplicationServices/ApplicationServices.h>

static int micStatus(void) { return (int)[AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio]; }
static void micRequest(void) { [AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL ok){}]; }
static int axTrusted(int prompt) {
	NSDictionary *o = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @(prompt != 0)};
	return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)o);
}
*/
import "C"

import "os/exec"

func Check() Status {
	mic := [...]string{"undetermined", "denied", "denied", "granted"}[C.micStatus()] // restricted counts as denied
	ax := "denied"
	if C.axTrusted(0) != 0 {
		ax = "granted"
	}
	return Status{Microphone: mic, Accessibility: ax}
}

// RequestMicrophone shows the system prompt if the user hasn't decided yet.
func RequestMicrophone() { C.micRequest() }

// RequestAccessibility shows the system prompt pointing to Privacy settings.
func RequestAccessibility() { C.axTrusted(1) }

// OpenSettings opens the Privacy pane: "microphone" or "accessibility".
func OpenSettings(pane string) error {
	anchor := map[string]string{"microphone": "Privacy_Microphone", "accessibility": "Privacy_Accessibility"}[pane]
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?"+anchor).Run()
}
