package transcription

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeStream replays a script of Decode results, one per Accept.
type fakeStream struct {
	script []struct {
		text     string
		endpoint bool
	}
	i     int
	tail  string
	reset int
}

func (f *fakeStream) Accept([]float32) {}
func (f *fakeStream) Decode() (string, bool) {
	s := f.script[f.i]
	f.i++
	return s.text, s.endpoint
}
func (f *fakeStream) Reset()         { f.reset++ }
func (f *fakeStream) Finish() string { return f.tail }
func (f *fakeStream) Close()         {}

func TestRunOrdersAndDedups(t *testing.T) {
	f := &fakeStream{tail: "THIRD"}
	for _, s := range []struct {
		text     string
		endpoint bool
	}{{"HELLO", false}, {"HELLO", false}, {"HELLO WORLD", false}, {"HELLO WORLD", true}, {"", false}, {"SECOND ONE", true}, {"THI", false}} {
		f.script = append(f.script, s)
	}
	in := make(chan []float32, len(f.script))
	for range f.script {
		in <- nil
	}
	close(in)
	out := make(chan Event, 32)
	Run(f, nil, in, out)

	var partials, finals []string
	for e := range out {
		if e.Final {
			finals = append(finals, e.Text)
		} else {
			partials = append(partials, e.Text)
		}
	}
	if got := strings.Join(finals, "|"); got != "Hello world|Second one|Third" {
		t.Fatalf("finals = %q", got)
	}
	if got := strings.Join(partials, "|"); got != "hello|hello world|thi" {
		t.Fatalf("partials = %q (repeated partials must be suppressed)", got)
	}
	if f.reset != 2 {
		t.Fatalf("reset %d times, want 2", f.reset)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"  I THINK  I'M READY ": "I think I'm ready",
		"":                      "",
		"Already Mixed":         "Already Mixed",
	} {
		if got := Normalize(in, nil); got != want {
			t.Errorf("Normalize(%q) = %q want %q", in, got, want)
		}
	}
}

// TestSherpaEndToEnd synthesizes speech with macOS `say` and runs the real model.
// Run with SAYKI_TEST_MODEL=/path/to/sherpa-onnx-streaming-zipformer-en-2023-06-26.
func TestSherpaEndToEnd(t *testing.T) {
	dir := os.Getenv("SAYKI_TEST_MODEL")
	if dir == "" {
		t.Skip("SAYKI_TEST_MODEL not set")
	}
	eng, err := NewSherpaTransducer(dir)
	if err != nil {
		t.Fatal(err)
	}
	endToEnd(t, eng)
}

// Run with SAYKI_TEST_PARAKEET=/path/to/sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8.
func TestParakeetEndToEnd(t *testing.T) {
	dir := os.Getenv("SAYKI_TEST_PARAKEET")
	if dir == "" {
		t.Skip("SAYKI_TEST_PARAKEET not set")
	}
	eng, err := NewParakeet(dir)
	if err != nil {
		t.Fatal(err)
	}
	endToEnd(t, eng)
}

func endToEnd(t *testing.T, eng Engine) {
	t.Helper()
	wav := filepath.Join(t.TempDir(), "s.wav")
	if err := exec.Command("say", "-o", wav, "--data-format=LEI16@16000", "Hello world, this is a test of offline dictation.").Run(); err != nil {
		t.Skip("say unavailable:", err)
	}
	raw, _ := os.ReadFile(wav)
	pcm := raw[44:] // shortcut: assumes canonical 44-byte WAV header, which `say` writes.
	samples := make([]float32, len(pcm)/2)
	for i := range samples {
		samples[i] = float32(int16(binary.LittleEndian.Uint16(pcm[2*i:]))) / 32768
	}

	defer eng.Close()
	s := eng.NewStream()
	defer s.Close()

	in := make(chan []float32)
	out := make(chan Event, 256)
	go func() {
		for i := 0; i < len(samples); i += 320 { // 20 ms chunks, like the mic
			in <- samples[i:min(i+320, len(samples))]
		}
		close(in)
	}()
	Run(s, nil, in, out)
	var finals []string
	for e := range out {
		if e.Final {
			finals = append(finals, e.Text)
		}
	}
	got := strings.ToLower(Join(finals))
	t.Logf("transcript: %q", got)
	for _, w := range []string{"hello", "world", "test", "dictation"} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in %q", w, got)
		}
	}
}
