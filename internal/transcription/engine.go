// Package transcription defines the pluggable speech-engine interface and the
// streaming pipeline that turns microphone samples into ordered text segments.
package transcription

// SampleRate is what the pipeline feeds engines; audio capture delivers this rate.
const SampleRate = 16000

// Engine is a loaded speech model. Implementations: Sherpa-ONNX (sherpa.go).
// A Whisper.cpp engine can be added by implementing these two interfaces.
type Engine interface {
	NewStream() Stream
	Close()
}

// Stream is one recognition session. Not safe for concurrent use.
type Stream interface {
	// Accept feeds mono float32 samples at SampleRate.
	Accept(samples []float32)
	// Decode runs the model on buffered audio and returns the current utterance
	// text and whether the engine detected an utterance boundary.
	Decode() (text string, endpoint bool)
	// Reset starts a new utterance after an endpoint.
	Reset()
	// Finish marks end of input and returns the remaining utterance text.
	Finish() string
	Close()
}

// Punctuator restores punctuation and casing on finalized text.
type Punctuator interface {
	Punctuate(text string) string
	Close()
}
