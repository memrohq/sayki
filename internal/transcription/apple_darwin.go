package transcription

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework Speech -framework AVFoundation
#import <Foundation/Foundation.h>
#import <Speech/Speech.h>
#import <AVFoundation/AVFoundation.h>
#include <stdlib.h>

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

static SFSpeechAudioBufferRecognitionRequest *newRequest(BOOL partial) {
	SFSpeechAudioBufferRecognitionRequest *req = [SFSpeechAudioBufferRecognitionRequest new];
	req.requiresOnDeviceRecognition = YES; // audio never leaves the Mac
	req.shouldReportPartialResults = partial;
	req.addsPunctuation = YES;
	return req;
}

static AVAudioPCMBuffer *pcm(const float *samples, int n) {
	static AVAudioFormat *fmt;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		fmt = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32 sampleRate:16000 channels:1 interleaved:NO];
	});
	AVAudioPCMBuffer *buf = [[AVAudioPCMBuffer alloc] initWithPCMFormat:fmt frameCapacity:(AVAudioFrameCount)n];
	memcpy(buf.floatChannelData[0], samples, sizeof(float) * n);
	buf.frameLength = (AVAudioFrameCount)n;
	return buf;
}

// Apple delivers one result per utterance (stretch of speech between pauses),
// each holding only that utterance's words, and later results revise the
// utterance they overlap. Utterances keeps them in order by word timestamps.
@interface Utterances : NSObject
@property(strong) NSMutableArray<NSString *> *texts;
@property(assign) NSTimeInterval lastStart, lastEnd;
@end
@implementation Utterances
- (instancetype)init { if ((self = [super init])) { _texts = [NSMutableArray array]; _lastStart = -1; } return self; }
- (void)add:(SFTranscription *)t {
	if (t.formattedString.length == 0) return;
	NSTimeInterval start = t.segments.firstObject.timestamp;
	NSTimeInterval end = t.segments.lastObject.timestamp + t.segments.lastObject.duration;
	// A result starting after the previous utterance ended is a new utterance;
	// anything overlapping it (same start, or a revision) replaces it.
	if (self.texts.count == 0 || start >= self.lastEnd - 0.05) {
		[self.texts addObject:t.formattedString];
	} else {
		self.texts[self.texts.count - 1] = t.formattedString;
	}
	self.lastStart = start;
	self.lastEnd = end;
}
// Joins utterances, ending one with a full stop when the next starts a new
// sentence (Apple doesn't punctuate the end of an utterance before a pause).
- (NSString *)joined {
	NSMutableString *out = [NSMutableString string];
	for (NSUInteger i = 0; i < self.texts.count; i++) {
		NSString *t = self.texts[i];
		[out appendString:t];
		if (i + 1 == self.texts.count) break;
		unichar last = [t characterAtIndex:t.length - 1], next = [self.texts[i + 1] characterAtIndex:0];
		if (![[NSCharacterSet punctuationCharacterSet] characterIsMember:last] && [[NSCharacterSet uppercaseLetterCharacterSet] characterIsMember:next])
			[out appendString:@"."];
		[out appendString:@" "];
	}
	return out;
}
@end

// Live preview: a streaming task whose partial results are shown while recording.
@interface SaykiPreview : NSObject
@property(strong) SFSpeechAudioBufferRecognitionRequest *request;
@property(strong) SFSpeechRecognitionTask *task;
@property(strong) Utterances *utterances;
@end
@implementation SaykiPreview
@end

static void *previewStart(void *rp) {
	SFSpeechRecognizer *r = (__bridge SFSpeechRecognizer *)rp;
	SaykiPreview *p = [SaykiPreview new];
	p.request = newRequest(YES);
	p.utterances = [Utterances new];
	__weak SaykiPreview *wp = p;
	p.task = [r recognitionTaskWithRequest:p.request resultHandler:^(SFSpeechRecognitionResult *res, NSError *err) {
		SaykiPreview *pp = wp;
		if (pp && res) { @synchronized (pp) { [pp.utterances add:res.bestTranscription]; } }
	}];
	return (__bridge_retained void *)p;
}

static void previewAccept(void *pp, const float *samples, int n) {
	SaykiPreview *p = (__bridge SaykiPreview *)pp;
	[p.request appendAudioPCMBuffer:pcm(samples, n)];
}

static char *previewText(void *pp) {
	SaykiPreview *p = (__bridge SaykiPreview *)pp;
	@synchronized (p) { return dupstr([p.utterances joined]); }
}

static void previewStop(void *pp) {
	SaykiPreview *p = (__bridge_transfer SaykiPreview *)pp;
	[p.task cancel];
}

// Final text: the whole recording in one request with partial results off, so
// only Apple's settled result for each utterance is used.
// *err receives a short error description (never the transcript), or NULL.
static char *appleTranscribe(void *rp, const float *samples, int n, char **err) {
	SFSpeechRecognizer *r = (__bridge SFSpeechRecognizer *)rp;
	SFSpeechAudioBufferRecognitionRequest *req = newRequest(NO);
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	Utterances *utterances = [Utterances new];
	__block NSString *failure = nil;
	__block BOOL signalled = NO;
	NSObject *lock = [NSObject new];
	SFSpeechRecognitionTask *task = [r recognitionTaskWithRequest:req resultHandler:^(SFSpeechRecognitionResult *res, NSError *e) {
		@synchronized (lock) {
			if (res) [utterances add:res.bestTranscription];
			if (e && e.code != 1110) failure = [NSString stringWithFormat:@"%@ %ld", e.domain, (long)e.code]; // 1110: no speech
			if ((res.isFinal || e) && !signalled) { signalled = YES; dispatch_semaphore_signal(done); }
		}
	}];
	for (int i = 0; i < n; i += 16000) [req appendAudioPCMBuffer:pcm(samples + i, MIN(16000, n - i))];
	[req endAudio];
	// Generous timeout: on-device recognition runs well faster than real time.
	long timedOut = dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, (int64_t)((10 + n / 16000.0) * NSEC_PER_SEC)));
	[task cancel];
	@synchronized (lock) {
		if (timedOut) failure = @"timed out";
		*err = failure ? dupstr(failure) : NULL;
		return dupstr([utterances joined]);
	}
}
*/
import "C"

import (
	"errors"
	"log/slog"
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

func (e *appleEngine) NewStream() Stream {
	return &appleStream{rec: e.rec, preview: C.previewStart(e.rec)}
}
func (e *appleEngine) Close() { C.appleRelease(e.rec) }

type appleStream struct {
	rec     unsafe.Pointer
	preview unsafe.Pointer
	audio   []float32 // whole recording, transcribed in one pass by Finish
}

func (s *appleStream) Accept(samples []float32) {
	if len(samples) == 0 {
		return
	}
	s.audio = append(s.audio, samples...)
	C.previewAccept(s.preview, (*C.float)(unsafe.Pointer(&samples[0])), C.int(len(samples)))
}

// Decode reports the live preview; Apple keeps one utterance per recording.
func (s *appleStream) Decode() (string, bool) { return take(C.previewText(s.preview)), false }
func (s *appleStream) Reset()                 {}

func (s *appleStream) Finish() string {
	if len(s.audio) == 0 {
		return ""
	}
	var cerr *C.char
	text := take(C.appleTranscribe(s.rec, (*C.float)(unsafe.Pointer(&s.audio[0])), C.int(len(s.audio)), &cerr))
	if cerr != nil {
		slog.Warn("apple speech", "err", take(cerr), "chars", len(text)) // never the text itself
	}
	return text
}

func (s *appleStream) Close() { C.previewStop(s.preview) }

func take(p *C.char) string {
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}
