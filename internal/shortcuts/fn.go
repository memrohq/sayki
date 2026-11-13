package shortcuts

import (
	"sync"
	"time"
)

// fnHold is how long Fn must be held alone before it counts as "pressed". A
// shorter tap still fires (pressed then released) on release; Fn used as a
// modifier (Fn+Delete, Fn+arrows) never fires at all.
const fnHold = 180 * time.Millisecond

// fnGate turns raw Fn key events into pressed/released callbacks. Every
// callback goes through one queue so a quick tap always arrives as
// pressed-then-released, in order.
type fnGate struct {
	hold   time.Duration
	mu     sync.Mutex
	down   bool // Fn currently held
	sent   bool // "pressed" already emitted for this hold
	chord  bool // another key was used with Fn, so this hold isn't ours
	closed bool
	timer  *time.Timer
	events chan bool
}

func newFnGate(hold time.Duration, emit func(pressed bool)) *fnGate {
	g := &fnGate{hold: hold, events: make(chan bool, 16)}
	go func() {
		for p := range g.events {
			emit(p)
		}
	}()
	return g
}

func (g *fnGate) key(down bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if down {
		if g.down {
			return
		}
		g.down, g.sent, g.chord = true, false, false
		g.timer = time.AfterFunc(g.hold, g.held)
		return
	}
	if !g.down {
		return
	}
	g.down = false
	g.timer.Stop()
	if g.chord && !g.sent {
		return
	}
	if !g.sent {
		g.send(true) // quick tap
	}
	g.send(false)
}

// other reports any other key or modifier activity while Fn may be held.
func (g *fnGate) other() {
	g.mu.Lock()
	if g.down && !g.sent {
		g.chord = true
	}
	g.mu.Unlock()
}

func (g *fnGate) held() {
	g.mu.Lock()
	if g.down && !g.chord && !g.sent {
		g.sent = true
		g.send(true)
	}
	g.mu.Unlock()
}

// send must hold mu. It never blocks the event tap; a full queue means the
// app is wedged anyway.
func (g *fnGate) send(p bool) {
	if g.closed {
		return
	}
	select {
	case g.events <- p:
	default:
	}
}

func (g *fnGate) close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.timer != nil {
		g.timer.Stop()
	}
	if !g.closed {
		g.closed = true
		close(g.events)
	}
}
