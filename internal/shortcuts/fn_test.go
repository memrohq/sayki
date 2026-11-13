package shortcuts

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func record(t *testing.T) (*fnGate, func() []bool) {
	var mu sync.Mutex
	var got []bool
	g := newFnGate(30*time.Millisecond, func(p bool) { mu.Lock(); got = append(got, p); mu.Unlock() })
	t.Cleanup(g.close)
	return g, func() []bool {
		time.Sleep(20 * time.Millisecond) // let the queue drain
		mu.Lock()
		defer mu.Unlock()
		return append([]bool(nil), got...)
	}
}

func TestFnTapFiresPressThenRelease(t *testing.T) {
	g, got := record(t)
	g.key(true)
	g.key(false)
	if want := []bool{true, false}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("got %v want %v", got(), want)
	}
}

func TestFnHoldPressesBeforeRelease(t *testing.T) {
	g, got := record(t)
	g.key(true)
	time.Sleep(60 * time.Millisecond)
	if want := []bool{true}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("while held: got %v want %v", got(), want)
	}
	g.key(false)
	if want := []bool{true, false}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("after release: got %v want %v", got(), want)
	}
}

func TestFnChordIsIgnored(t *testing.T) {
	g, got := record(t)
	g.key(true)
	g.other() // e.g. Fn+Delete
	time.Sleep(60 * time.Millisecond)
	g.key(false)
	if len(got()) != 0 {
		t.Fatalf("chord fired %v", got())
	}
}

func TestParseSolo(t *testing.T) {
	for _, s := range []string{"fn", "ctrl", "rightoption", "rightcmd", "Fn"} {
		if a, err := Parse(s); err != nil || a.Solo == "" {
			t.Errorf("Parse(%q) = %+v %v", s, a, err)
		}
	}
}
