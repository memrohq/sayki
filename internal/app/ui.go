package app

import (
	"github.com/getvybe/sayki/internal/injection"
	"github.com/getvybe/sayki/internal/models"
	"github.com/getvybe/sayki/internal/permissions"
	"github.com/getvybe/sayki/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	overlayW = 440
	overlayH = 96
	// overlayDragH: top strip (bars row) that drags the overlay natively; must stop
	// above the caption row in Overlay.tsx, which holds the Done/✕ buttons.
	overlayDragH = 52
)

var (
	eventsWindowClosing = events.Common.WindowClosing
	eventsAppStarted    = events.Common.ApplicationStarted
)

func (a *App) ShowSettings() {
	application.InvokeAsync(func() {
		a.prefs.Show()
		a.prefs.Focus() // activates Sayki; fine here, the user asked for this window
	})
}

// showOverlay positions the overlay on the screen under the mouse and shows it without
// activating Sayki. force shows it even when the user hid the overlay (errors).
func (a *App) showOverlay(force bool) {
	a.mu.Lock()
	pos, placed, last := a.cfg.Overlay, a.placed, a.overlayScreen
	show := pos != settings.OverlayHidden || force
	a.mu.Unlock()
	if !show {
		return
	}
	cx, cy := injection.Cursor()
	application.InvokeAsync(func() {
		// The screen the user is working on (under the mouse), not always the main one.
		scr := a.wails.Screen.ScreenNearestDipPoint(application.Point{X: cx, Y: cy})
		if scr == nil {
			scr = a.wails.Screen.GetPrimary()
		}
		a.mu.Lock()
		a.placed = true
		if scr != nil {
			a.overlayScreen = scr.ID
		}
		a.mu.Unlock()
		// Keep a dragged position only while the user stays on that screen.
		if scr != nil && (!placed || scr.ID != last) {
			wa := scr.WorkArea
			x := wa.X + (wa.Width-overlayW)/2
			y := wa.Y + wa.Height - overlayH - 24
			if pos == settings.OverlayTop {
				y = wa.Y + 12
			}
			a.overlay.SetPosition(x, y)
		}
		a.overlay.Show()
	})
}

func (a *App) hideOverlay() { application.InvokeAsync(func() { a.overlay.Hide() }) }

func (a *App) rebuildMenu() {
	// SetMenu waits on the main thread, which is blocked in ServiceShutdown while we quit.
	if a.tray == nil || a.shutdown.Err() != nil {
		return
	}
	a.mu.Lock()
	cfg, state := a.cfg, a.status.State
	a.mu.Unlock()
	mics, _ := a.audio.Microphones()
	perm := permissions.Check()

	m := a.wails.NewMenu()
	label := "Start Dictation"
	if state == StateRecording {
		label = "Stop Dictation"
	}
	m.Add(label + "    " + ShortcutLabel(cfg.Shortcut)).OnClick(func(*application.Context) { a.ToggleRecording() })
	m.AddSeparator()

	micMenu := m.AddSubmenu("Microphone")
	micMenu.AddRadio("System Default", cfg.MicrophoneID == "").OnClick(func(*application.Context) {
		a.update(func(s *settings.Settings) { s.MicrophoneID = "" })
	})
	for _, d := range mics {
		micMenu.AddRadio(d.Name, cfg.MicrophoneID == d.ID).OnClick(func(*application.Context) {
			a.update(func(s *settings.Settings) { s.MicrophoneID = d.ID })
		})
	}

	langMenu := m.AddSubmenu("Language")
	for _, mod := range models.Catalog {
		if mod.Kind != models.KindASR || !a.models.Installed(mod.ID) {
			continue
		}
		langMenu.AddRadio(mod.Name, cfg.ModelID == mod.ID).OnClick(func(*application.Context) {
			a.update(func(s *settings.Settings) { s.ModelID = mod.ID })
		})
	}
	langMenu.AddSeparator()
	langMenu.Add("Manage Models…").OnClick(func(*application.Context) { a.ShowSettings() })

	modeMenu := m.AddSubmenu("Mode")
	modeMenu.AddRadio("Toggle (press to start/stop)", cfg.Mode == settings.ModeToggle).OnClick(func(*application.Context) {
		a.update(func(s *settings.Settings) { s.Mode = settings.ModeToggle })
	})
	modeMenu.AddRadio("Push to talk (hold)", cfg.Mode == settings.ModePushToTalk).OnClick(func(*application.Context) {
		a.update(func(s *settings.Settings) { s.Mode = settings.ModePushToTalk })
	})
	m.AddSeparator()

	mark := map[string]string{"granted": "✓", "denied": "✗", "undetermined": "?", "unsupported": "–"}
	m.Add("Microphone access " + mark[perm.Microphone]).OnClick(func(*application.Context) { permissions.OpenSettings("microphone") })
	m.Add("Accessibility access " + mark[perm.Accessibility]).OnClick(func(*application.Context) {
		permissions.RequestAccessibility()
		permissions.OpenSettings("accessibility")
	})
	m.AddSeparator()
	m.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { a.ShowSettings() })
	m.Add("Quit Sayki").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { a.wails.Quit() })
	a.tray.SetMenu(m)
}
