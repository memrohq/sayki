package injection

import "testing"

func TestSanitizeNeverSubmits(t *testing.T) {
	if got := Sanitize("ls -la\n\rrm\x1b[A\tok\x7f"); got != "ls -la  rm[A ok" {
		t.Fatalf("got %q", got)
	}
}
