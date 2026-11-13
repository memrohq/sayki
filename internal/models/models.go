// Package models manages locally installed Sherpa-ONNX models. Downloads only
// happen when the user explicitly asks; recognition never touches the network.
package models

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const releases = "https://github.com/k2-fsa/sherpa-onnx/releases/download/"

type Kind string

const (
	KindASR   Kind = "asr"
	KindPunct Kind = "punct"
)

type Model struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Kind     Kind   `json:"kind"`
	URL      string `json:"-"`
	Bytes    int64  `json:"bytes"`  // download size
	Engine   string `json:"engine"` // which transcription engine loads it
}

// Engines. Streaming Zipformer is the default; Parakeet transcribes the whole
// recording when it ends; Apple is macOS's built-in on-device recognizer.
const (
	EngineZipformer = ""
	EngineParakeet  = "parakeet"
	EngineApple     = "apple"
)

// AppleID is the built-in macOS recognizer: nothing to download.
const AppleID = "apple"

// PunctID is the English punctuation + casing model.
const PunctID = "punct-en"

var Catalog = []Model{
	{AppleID, "Apple speech (built into macOS)", "en", KindASR, "", 0, EngineApple},
	{"en-parakeet", "English (NVIDIA Parakeet, most accurate)", "en", KindASR, releases + "asr-models/sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8.tar.bz2", 482440461, EngineParakeet},
	{"en-zipformer", "English (Zipformer, accurate)", "en", KindASR, releases + "asr-models/sherpa-onnx-streaming-zipformer-en-2023-06-26.tar.bz2", 310414022, EngineZipformer},
	{"en-zipformer-small", "English (Zipformer 20M, fast)", "en", KindASR, releases + "asr-models/sherpa-onnx-streaming-zipformer-en-20M-2023-02-17.tar.bz2", 127887156, EngineZipformer},
	{"zh-en-zipformer", "Chinese + English (Zipformer)", "zh-en", KindASR, releases + "asr-models/sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20.tar.bz2", 511274346, EngineZipformer},
	{"fr-zipformer", "French (Zipformer)", "fr", KindASR, releases + "asr-models/sherpa-onnx-streaming-zipformer-fr-2023-04-14.tar.bz2", 398444115, EngineZipformer},
	{"ko-zipformer", "Korean (Zipformer)", "ko", KindASR, releases + "asr-models/sherpa-onnx-streaming-zipformer-korean-2024-06-16.tar.bz2", 418218652, EngineZipformer},
	{PunctID, "English punctuation & casing", "en", KindPunct, releases + "punctuation-models/sherpa-onnx-online-punct-en-2024-08-06.tar.bz2", 30667839, EngineZipformer},
}

func Find(id string) (Model, bool) {
	for _, m := range Catalog {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

type Manager struct{ Root string }

func (m Manager) Dir(id string) string { return filepath.Join(m.Root, id) }

func (m Manager) Installed(id string) bool {
	if id == AppleID {
		return true // part of macOS
	}
	_, err := os.Stat(filepath.Join(m.Dir(id), ".complete"))
	return err == nil
}

func (m Manager) Delete(id string) error {
	if _, ok := Find(id); !ok {
		return fmt.Errorf("unknown model %q", id)
	}
	if id == AppleID {
		return fmt.Errorf("Apple speech is part of macOS and can't be removed")
	}
	return os.RemoveAll(m.Dir(id))
}

// Download fetches and extracts a catalog model. progress receives bytes downloaded.
// Extraction goes to a temp dir that is renamed into place only on success.
func (m Manager) Download(ctx context.Context, id string, progress func(done, total int64)) error {
	model, ok := Find(id)
	if !ok {
		return fmt.Errorf("unknown model %q", id)
	}
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, model.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", id, resp.Status)
	}
	total := resp.ContentLength
	if total <= 0 {
		total = model.Bytes
	}

	tmp, err := os.MkdirTemp(m.Root, ".dl-"+id+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	body := &countingReader{r: resp.Body, total: total, fn: progress}
	if err := extract(tar.NewReader(bzip2.NewReader(body)), tmp); err != nil {
		return fmt.Errorf("extract %s: %w", id, err)
	}
	// Archives contain one top-level folder; flatten it.
	src := tmp
	if ents, _ := os.ReadDir(tmp); len(ents) == 1 && ents[0].IsDir() {
		src = filepath.Join(tmp, ents[0].Name())
	}
	os.RemoveAll(filepath.Join(src, "test_wavs"))
	if err := os.WriteFile(filepath.Join(src, ".complete"), nil, 0o644); err != nil {
		return err
	}
	if err := os.RemoveAll(m.Dir(id)); err != nil {
		return err
	}
	return os.Rename(src, m.Dir(id))
}

func extract(tr *tar.Reader, dest string) error {
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, h.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %q", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		} // symlinks and other entries are skipped on purpose
	}
}

type countingReader struct {
	r           io.Reader
	done, total int64
	lastPct     int64
	fn          func(done, total int64)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.done += int64(n)
	if c.fn != nil && c.total > 0 {
		if pct := c.done * 100 / c.total; pct != c.lastPct {
			c.lastPct = pct
			c.fn(c.done, c.total)
		}
	}
	return n, err
}
