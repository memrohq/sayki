package shortcuts

import (
	"testing"

	"github.com/getvybe/sayki/internal/settings"
)

func TestParse(t *testing.T) {
	a, err := Parse("ctrl+option+Space")
	if err != nil || !a.Ctrl || !a.Option || a.Cmd || a.Code != "Space" {
		t.Fatalf("got %+v %v", a, err)
	}
	for _, bad := range []string{"Space", "shift+KeyA", "ctrl+Nope", "hyper+KeyA", ""} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
	if a, err := Parse("ctrl+option+space"); err != nil || a.Code != "Space" {
		t.Errorf("lowercase key: got %+v %v", a, err)
	}
	if _, err := Parse("F13"); err != nil {
		t.Errorf("bare F-key rejected: %v", err)
	}
}

func TestDefaultShortcutParses(t *testing.T) {
	if _, err := Parse(settings.Defaults().Shortcut); err != nil {
		t.Fatal(err)
	}
}
