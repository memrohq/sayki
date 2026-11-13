package models

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "../evil", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644})
	tw.Write([]byte("x"))
	tw.Close()
	err := extract(tar.NewReader(&buf), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
}

func TestCatalogIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range Catalog {
		if seen[m.ID] {
			t.Fatalf("duplicate id %s", m.ID)
		}
		seen[m.ID] = true
	}
}

// Real network download of the smallest model. Run with SAYKI_TEST_DOWNLOAD=1.
func TestDownloadPunct(t *testing.T) {
	if os.Getenv("SAYKI_TEST_DOWNLOAD") == "" {
		t.Skip("SAYKI_TEST_DOWNLOAD not set")
	}
	m := Manager{Root: t.TempDir()}
	if err := m.Download(context.Background(), PunctID, nil); err != nil {
		t.Fatal(err)
	}
	if !m.Installed(PunctID) {
		t.Fatal("not marked installed")
	}
	if _, err := os.Stat(filepath.Join(m.Dir(PunctID), "bpe.vocab")); err != nil {
		t.Fatal(err)
	}
}
