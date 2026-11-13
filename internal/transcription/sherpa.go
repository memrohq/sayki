package transcription

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

type sherpaEngine struct{ rec *sherpa.OnlineRecognizer }

// NewSherpaTransducer loads a streaming transducer model directory
// (encoder*.onnx, decoder*.onnx, joiner*.onnx, tokens.txt), preferring int8 files.
func NewSherpaTransducer(dir string) (Engine, error) {
	pick := func(prefix string) (string, error) {
		matches, _ := filepath.Glob(filepath.Join(dir, prefix+"*.onnx"))
		if len(matches) == 0 {
			return "", fmt.Errorf("model %s: no %s*.onnx", dir, prefix)
		}
		for _, m := range matches {
			if strings.Contains(m, ".int8.") {
				return m, nil
			}
		}
		return matches[0], nil
	}
	enc, err := pick("encoder")
	if err != nil {
		return nil, err
	}
	dec, err := pick("decoder")
	if err != nil {
		return nil, err
	}
	join, err := pick("joiner")
	if err != nil {
		return nil, err
	}
	tokens := filepath.Join(dir, "tokens.txt")
	if _, err := os.Stat(tokens); err != nil {
		return nil, fmt.Errorf("model %s: %w", dir, err)
	}

	c := sherpa.OnlineRecognizerConfig{}
	c.FeatConfig = sherpa.FeatureConfig{SampleRate: SampleRate, FeatureDim: 80}
	c.ModelConfig.Transducer = sherpa.OnlineTransducerModelConfig{Encoder: enc, Decoder: dec, Joiner: join}
	c.ModelConfig.Tokens = tokens
	c.ModelConfig.NumThreads = min(2, runtime.NumCPU())
	c.ModelConfig.Provider = "cpu"
	c.DecodingMethod = "greedy_search"
	c.EnableEndpoint = 1
	c.Rule1MinTrailingSilence = 2.4
	c.Rule2MinTrailingSilence = 0.8
	c.Rule3MinUtteranceLength = 20

	rec := sherpa.NewOnlineRecognizer(&c)
	if rec == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to load model in %s", dir)
	}
	return &sherpaEngine{rec: rec}, nil
}

func (e *sherpaEngine) NewStream() Stream {
	return &sherpaStream{rec: e.rec, s: sherpa.NewOnlineStream(e.rec)}
}

func (e *sherpaEngine) Close() { sherpa.DeleteOnlineRecognizer(e.rec) }

type sherpaStream struct {
	rec *sherpa.OnlineRecognizer
	s   *sherpa.OnlineStream
}

func (s *sherpaStream) Accept(samples []float32) { s.s.AcceptWaveform(SampleRate, samples) }

func (s *sherpaStream) Decode() (string, bool) {
	for s.rec.IsReady(s.s) {
		s.rec.Decode(s.s)
	}
	return s.rec.GetResult(s.s).Text, s.rec.IsEndpoint(s.s)
}

func (s *sherpaStream) Reset() { s.rec.Reset(s.s) }

func (s *sherpaStream) Finish() string {
	// Trailing silence lets the model emit the last word(s) before input ends.
	s.s.AcceptWaveform(SampleRate, make([]float32, SampleRate*3/10))
	s.s.InputFinished()
	text, _ := s.Decode()
	return text
}

func (s *sherpaStream) Close() { sherpa.DeleteOnlineStream(s.s) }

type sherpaPunct struct{ p *sherpa.OnlinePunctuation }

// NewSherpaPunctuator loads an online CNN-BiLSTM punctuation+casing model dir.
func NewSherpaPunctuator(dir string) (Punctuator, error) {
	model, _ := filepath.Glob(filepath.Join(dir, "model*.onnx"))
	vocab := filepath.Join(dir, "bpe.vocab")
	if len(model) == 0 {
		return nil, fmt.Errorf("punctuation model %s: no model*.onnx", dir)
	}
	pick := model[0]
	for _, m := range model {
		if strings.Contains(m, ".int8.") {
			pick = m
		}
	}
	c := sherpa.OnlinePunctuationConfig{}
	c.Model = sherpa.OnlinePunctuationModelConfig{CnnBilstm: pick, BpeVocab: vocab, NumThreads: 1, Provider: "cpu"}
	p := sherpa.NewOnlinePunctuation(&c)
	if p == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to load punctuation model in %s", dir)
	}
	return &sherpaPunct{p: p}, nil
}

func (p *sherpaPunct) Punctuate(text string) string { return p.p.AddPunct(text) }
func (p *sherpaPunct) Close()                       { sherpa.DeleteOnlinePunctuation(p.p) }
