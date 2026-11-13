package transcription

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework Speech -framework AVFoundation
#import <Foundation/Foundation.h>
#import <Speech/Speech.h>
#import <AVFoundation/AVFoundation.h>
#include <stdlib.h>

@interface SaykiSession : NSObject
@property(strong) SFSpeechAudioBufferRecognitionRequest *request;
@property(strong) SFSpeechRecognitionTask *task;
@property(copy) NSString *text;
@property(assign) BOOL done;
@property(strong) dispatch_semaphore_t finished;
@end
@implementation SaykiSession
@end

static char *dupstr(NSString *s) { return strdup(s ? s.UTF8String : ""); }

// 0 authorized, 1 denied/restricted. Asks the user on first use.
static int appleAuthorize(void) {
	SFSpeechRecognizerAuthorizationStatus st = [SFSpeechRecognizer authorizationStatus];
	if (st == SFSpeechRecognizerAuthorizationStatusNotDetermined) {
		dispatch_semaphore_t sem = dispatch_semaphore_create(0);
		__block SFSpeechRecognizerAuthorizationStatus got = st;
		[SFSpeechRecognizer requestAuthorization:^(SFSpeechRecognizerAuthorizationStatus s) { got = s; dispatch_semaphore_signal(sem); }];
		dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
		st = got;
	}
	return st == SFSpeechRecognizerAuthorizationStatusAuthorized ? 0 : 1;
}

// Returns a retained recognizer for the system language (falling back to US
// English) that can run on-device, or NULL when none is available.
static void *appleRecognizer(void) {
	for (NSLocale *loc in @[[NSLocale currentLocale], [NSLocale localeWithLocaleIdentifier:@"en-US"]]) {
		SFSpeechRecognizer *r = [[SFSpeechRecognizer alloc] initWithLocale:loc];
		if (r && r.supportsOnDeviceRecognition) {
			r.queue = [[NSOperationQueue alloc] init]; // results off the main thread
			return (__bridge_retained void *)r;
		}
	}
	return NULL;
}

static void appleRelease(void *p) { if (p) CFBridgingRelease(p); }

static void *appleStart(void *rp) {
	SFSpeechRecognizer *r = (__bridge SFSpeechRecognizer *)rp;
	SaykiSession *s = [SaykiSession new];
	s.finished = dispatch_semaphore_create(0);
	s.request = [SFSpeechAudioBufferRecognitionRequest new];
	s.request.requiresOnDeviceRecognition = YES; // audio never leaves the Mac
	s.request.shouldReportPartialResults = YES;
	s.request.addsPunctuation = YES;
	__weak SaykiSession *ws = s;
	s.task = [r recognitionTaskWithRequest:s.request resultHandler:^(SFSpeechRecognitionResult *res, NSError *err) {
		SaykiSession *ss = ws;
		if (!ss) return;
		@synchronized (ss) {
			if (res) ss.text = res.bestTranscription.formattedString;
			if ((res.isFinal || err) && !ss.done) {
				ss.done = YES;
				dispatch_semaphore_signal(ss.finished);
			}
		}
	}];
	return (__bridge_retained void *)s;
}

static void appleAccept(void *sp, const float *samples, int n) {
	static AVAudioFormat *fmt;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		fmt = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32 sampleRate:16000 channels:1 interleaved:NO];
	});
	SaykiSession *s = (__bridge SaykiSession *)sp;
	AVAudioPCMBuffer *buf = [[AVAudioPCMBuffer alloc] initWithPCMFormat:fmt frameCapacity:(AVAudioFrameCount)n];
	memcpy(buf.floatChannelData[0], samples, sizeof(float) * n);
	buf.frameLength = (AVAudioFrameCount)n;
	[s.request appendAudioPCMBuffer:buf];
}

static char *appleText(void *sp) {
	SaykiSession *s = (__bridge SaykiSession *)sp;
	@synchronized (s) { return dupstr(s.text); }
}

static char *appleFinish(void *sp) {
	SaykiSession *s = (__bridge SaykiSession *)sp;
	[s.request endAudio];
	dispatch_semaphore_wait(s.finished, dispatch_time(DISPATCH_TIME_NOW, 5 * NSEC_PER_SEC));
	return appleText(sp);
}

static void appleStop(void *sp) {
	SaykiSession *s = (__bridge_transfer SaykiSession *)sp;
	[s.task cancel];
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

var (
	ErrSpeechDenied      = errors.New("Speech Recognition is off for Sayki: enable it in System Settings → Privacy & Security → Speech Recognition")
	ErrSpeechUnavailable = errors.New("on-device Apple speech isn't available: turn on Dictation in System Settings → Keyboard so macOS downloads it, or pick another model")
)

type appleEngine struct{ rec unsafe.Pointer }

// NewApple uses macOS's built-in on-device speech recognizer: nothing to download.
func NewApple() (Engine, error) {
	if C.appleAuthorize() != 0 {
		return nil, ErrSpeechDenied
	}
	rec := C.appleRecognizer()
	if rec == nil {
		return nil, ErrSpeechUnavailable
	}
	return &appleEngine{rec: rec}, nil
}

func (e *appleEngine) NewStream() Stream { return &appleStream{s: C.appleStart(e.rec)} }
func (e *appleEngine) Close()            { C.appleRelease(e.rec) }

type appleStream struct{ s unsafe.Pointer }

func (s *appleStream) Accept(samples []float32) {
	if len(samples) > 0 {
		C.appleAccept(s.s, (*C.float)(unsafe.Pointer(&samples[0])), C.int(len(samples)))
	}
}

// Decode reports Apple's latest partial result; Apple keeps one utterance per recording.
func (s *appleStream) Decode() (string, bool) { return take(C.appleText(s.s)), false }
func (s *appleStream) Reset()                 {}
func (s *appleStream) Finish() string         { return take(C.appleFinish(s.s)) }
func (s *appleStream) Close()                 { C.appleStop(s.s) }

func take(p *C.char) string {
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}
