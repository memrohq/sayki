package settings

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	got, err := s.Load()
	if err != nil || got != Defaults() {
		t.Fatalf("fresh load = %+v, %v", got, err)
	}
	want := Defaults()
	want.Mode = ModePushToTalk
	want.MicrophoneID = "abc"
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	if got, _ = s.Load(); got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	bad := want
	bad.Mode = "nope"
	if s.Save(bad) == nil {
		t.Fatal("invalid mode accepted")
	}

	s.AddHistory("one")
	s.AddHistory("two")
	h, _ := s.History(10)
	if len(h) != 2 || h[0].Text != "two" {
		t.Fatalf("history %+v", h)
	}
	if err := s.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	if h, _ = s.History(10); len(h) != 0 {
		t.Fatalf("history not cleared: %+v", h)
	}
}
