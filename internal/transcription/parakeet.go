package transcription

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// Parakeet is not a streaming model: it transcribes the whole recording at once,
// which is what makes it accurate. While recording, a background preview pass
// re-decodes the audio so far every previewEvery, but only for the first
// previewMax: each pass costs more as audio grows. Previews run off the pipeline
// goroutine so a slow pass can never stall the mic and drop audio.
const (
	previewEvery = 2 * SampleRate
	previewMax   = 20 * SampleRate
)

type parakeetEngine struct{ rec *sherpa.OfflineRecognizer }

// NewParakeet loads an NVIDIA NeMo Parakeet TDT transducer exported by sherpa-onnx.
func NewParakeet(dir string) (Engine, error) {
	file := func(name string) (string, error) {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("model %s: %w", dir, err)
		}
		return p, nil
	}
	c := sherpa.OfflineRecognizerConfig{}
	c.FeatConfig = sherpa.FeatureConfig{SampleRate: SampleRate, FeatureDim: 80}
	var err error
	t := &c.ModelConfig.Transducer
	for _, f := range []struct {
		dst  *string
		name string
	}{{&t.Encoder, "encoder.int8.onnx"}, {&t.Decoder, "decoder.int8.onnx"}, {&t.Joiner, "joiner.int8.onnx"}, {&c.ModelConfig.Tokens, "tokens.txt"}} {
		if *f.dst, err = file(f.name); err != nil {
			return nil, err
		}
	}
	c.ModelConfig.ModelType = "nemo_transducer"
	c.ModelConfig.NumThreads = min(4, runtime.NumCPU())
	c.ModelConfig.Provider = "cpu"
	c.DecodingMethod = "greedy_search"

	rec := sherpa.NewOfflineRecognizer(&c)
	if rec == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to load Parakeet model in %s", dir)
	}
	return &parakeetEngine{rec: rec}, nil
}

func (e *parakeetEngine) NewStream() Stream { return &parakeetStream{rec: e.rec} }
func (e *parakeetEngine) Close()            { sherpa.DeleteOfflineRecognizer(e.rec) }

type parakeetStream struct {
	rec       *sherpa.OfflineRecognizer
	audio     []float32 // only appended to, so earlier prefixes are safe to share
	decodedAt int       // len(audio) when the last preview started
	busy      sync.WaitGroup

	mu      sync.Mutex
	running bool
	preview string
}

func (s *parakeetStream) Accept(samples []float32) { s.audio = append(s.audio, samples...) }

func (s *parakeetStream) Decode() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running && len(s.audio) <= previewMax && len(s.audio)-s.decodedAt >= previewEvery {
		s.running, s.decodedAt = true, len(s.audio)
		audio := s.audio[:len(s.audio):len(s.audio)]
		s.busy.Add(1)
		go func() {
			defer s.busy.Done()
			text := s.transcribe(audio)
			s.mu.Lock()
			s.preview, s.running = text, false
			s.mu.Unlock()
		}()
	}
	return s.preview, false // one utterance per recording; Finish gives the final text
}

func (s *parakeetStream) Reset() {}

func (s *parakeetStream) Finish() string {
	s.busy.Wait() // don't compete with a preview pass for the CPU
	if len(s.audio) == 0 {
		return ""
	}
	return s.transcribe(s.audio)
}

func (s *parakeetStream) Close() { s.busy.Wait() }

func (s *parakeetStream) transcribe(audio []float32) string {
	st := sherpa.NewOfflineStream(s.rec)
	defer sherpa.DeleteOfflineStream(st)
	st.AcceptWaveform(SampleRate, audio)
	s.rec.Decode(st)
	return st.GetResult().Text
}
