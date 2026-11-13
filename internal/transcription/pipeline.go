package transcription

import (
	"strings"
	"unicode"
)

// Event is emitted by Run. Partial events replace the previous partial for the
// current utterance; Final events are emitted exactly once per utterance, in order.
type Event struct {
	Text  string
	Final bool
}

// Run consumes audio until in is closed, then flushes the engine and closes out.
// Everything happens on this one goroutine, which is what keeps segment order
// stable and prevents a segment being finalized twice.
func Run(s Stream, p Punctuator, in <-chan []float32, out chan<- Event) {
	defer close(out)
	last := ""
	final := func(text string) {
		if text = Normalize(text, p); text != "" {
			out <- Event{Text: text, Final: true}
		}
		last = ""
	}
	for chunk := range in {
		s.Accept(chunk)
		text, endpoint := s.Decode()
		if endpoint {
			final(text)
			s.Reset()
			continue
		}
		if text != last {
			last = text
			out <- Event{Text: strings.ToLower(text)}
		}
	}
	final(s.Finish())
}

// Normalize cleans up raw model output. Zipformer models emit upper-case text
// without punctuation, so without a punctuator we lower-case and capitalize.
func Normalize(text string, p Punctuator) string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	if text != strings.ToUpper(text) {
		lower = text // model produced mixed case already; keep it
	}
	if p != nil {
		return p.Punctuate(lower)
	}
	words := strings.Fields(lower)
	for i, w := range words {
		if w == "i" || strings.HasPrefix(w, "i'") {
			words[i] = "I" + w[1:]
		}
	}
	r := []rune(strings.Join(words, " "))
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Join concatenates finalized segments with single spaces.
func Join(segments []string) string { return strings.Join(segments, " ") }
