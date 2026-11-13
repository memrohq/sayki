// Package app wires capture → transcription → injection together and exposes
// the API the React frontend calls through Wails bindings.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/memrohq/sayki/internal/audio"
	"github.com/memrohq/sayki/internal/injection"
	"github.com/memrohq/sayki/internal/install"
	"github.com/memrohq/sayki/internal/models"
	"github.com/memrohq/sayki/internal/permissions"
	"github.com/memrohq/sayki/internal/settings"
	"github.com/memrohq/sayki/internal/shortcuts"
	"github.com/memrohq/sayki/internal/sound"
	"github.com/memrohq/sayki/internal/transcription"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Events emitted to the frontend.
const (
	EvState      = "sayki:state"      // Status
	EvTranscript = "sayki:transcript" // string: committed text + live partial
	EvLevel      = "sayki:level"      // float32 RMS, ~20 Hz while recording
	EvDownload   = "sayki:download"   // DownloadProgress
	EvSettings   = "sayki:settings"   // settings.Settings, when changed outside the settings window
)

const (
	StateIdle      = "idle"
	StateLoading   = "loading"
	StateRecording = "recording"
	StateFinishing = "finishing"
	StateError     = "error"
)

type Status struct {
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
	Language  string `json:"language"`
	StartedAt int64  `json:"startedAt,omitempty"` // unix ms
}

type ModelInfo struct {
	models.Model
	Installed   bool `json:"installed"`
	Downloading bool `json:"downloading"`
}

type DownloadProgress struct {
	ID       string `json:"id"`
	Done     int64  `json:"done"`
	Total    int64  `json:"total"`
	Finished bool   `json:"finished"`
	Error    string `json:"error,omitempty"`
}

type session struct {
	cap      *audio.Capture
	target   injection.Target
	started  time.Time
	cancel   bool
	finished chan struct{}
}

type App struct {
	wails    *application.App
	overlay  *application.WebviewWindow
	prefs    *application.WebviewWindow
	tray     *application.SystemTray
	store    *settings.Store
	models   models.Manager
	audio    *audio.System
	shutdown context.Context
	stop     context.CancelFunc

	mu            sync.Mutex
	cfg           settings.Settings
	status        Status
	engine        transcription.Engine
	engineID      string
	punct         transcription.Punctuator
	sess          *session
	downloads     map[string]context.CancelFunc
	errTimer      *time.Timer
	placed        bool   // overlay positioned; later shows keep where the user dragged it
	overlayScreen string // screen ID the overlay was last shown on
}

func New(dataDir string) (*App, error) {
	store, err := settings.Open(filepath.Join(dataDir, "sayki.db"))
	if err != nil {
		return nil, err
	}
	cfg, err := store.Load()
	if err != nil {
		slog.Warn("settings unreadable, using defaults", "err", err)
	}
	snd, err := audio.NewSystem()
	if err != nil {
		store.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		store: store, audio: snd, cfg: cfg, shutdown: ctx, stop: cancel,
		models:    models.Manager{Root: filepath.Join(dataDir, "models")},
		status:    Status{State: StateIdle},
		downloads: map[string]context.CancelFunc{},
	}, nil
}

// Attach creates the windows and tray once the Wails app exists.
func (a *App) Attach(w *application.App, trayIcon []byte) {
	a.wails = w
	a.overlay = w.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "overlay", URL: "/#/overlay",
		Width: overlayW, Height: overlayH,
		Frameless: true, Hidden: true, DisableResize: true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent, // BackgroundType alone doesn't clear the window on macOS
			// Non-activating panel: showing it never takes focus from the target app.
			WindowClass:             application.MacWindowClassPanel,
			PanelPreferences:        application.MacPanelPreferences{NonActivating: true},
			WindowLevel:             application.MacWindowLevelStatus,
			CollectionBehavior:      application.MacWindowCollectionBehaviorCanJoinAllSpaces | application.MacWindowCollectionBehaviorFullScreenAuxiliary,
			DisableShadow:           true,
			InvisibleTitleBarHeight: overlayDragH,                          // drag starts inside mouse-down, the only time AppKit honours it
			CornerType:              application.MacWindowCornerTypeSquare, // borderless; the card draws its own corners
			TitleBar:                application.MacTitleBar{AppearsTransparent: true, Hide: true},
		},
	})
	a.prefs = w.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "settings", Title: "Sayki", URL: "/#/settings",
		Width: 720, Height: 560, MinWidth: 640, MinHeight: 480, Hidden: true,
		InitialPosition: application.WindowCentered,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
	})
	a.prefs.RegisterHook(eventsWindowClosing, func(e *application.WindowEvent) {
		a.prefs.Hide()
		e.Cancel()
	})

	a.tray = w.SystemTray.New()
	a.tray.SetTemplateIcon(trayIcon)
	a.tray.SetTooltip("Sayki")
	a.rebuildMenu()

	w.Event.OnApplicationEvent(eventsAppStarted, func(*application.ApplicationEvent) { go a.started() })
}

func (a *App) started() {
	var moved bool
	application.InvokeSync(func() {
		var err error
		if moved, err = install.OfferMove(); err != nil {
			slog.Warn("move to Applications", "err", err)
		}
	})
	if moved {
		slog.Info("moved to Applications; relaunching")
		a.wails.Quit()
		return
	}
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if err := a.registerShortcut(cfg.Shortcut); err != nil {
		slog.Error("shortcut", "err", err)
	} else {
		slog.Info("shortcut registered", "shortcut", cfg.Shortcut)
	}
	p := permissions.Check()
	slog.Info("permissions", "microphone", p.Microphone, "accessibility", p.Accessibility)
	if !a.models.Installed(cfg.ModelID) || p.Microphone != "granted" || p.Accessibility != "granted" {
		a.ShowSettings() // first-run onboarding
		return
	}
	go a.ensureEngine() // warm the model so the first dictation starts instantly
}

// Shutdown is called by Wails on quit.
func (a *App) ServiceShutdown() error {
	a.stop()
	shortcuts.Unregister()
	a.mu.Lock()
	s := a.sess
	a.mu.Unlock()
	if s != nil {
		a.Cancel()
		<-s.finished
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.engine != nil {
		a.engine.Close()
	}
	if a.punct != nil {
		a.punct.Close()
	}
	a.audio.Close()
	return a.store.Close()
}

func (a *App) registerShortcut(s string) error {
	acc, err := shortcuts.Parse(s)
	if err != nil {
		return err
	}
	return shortcuts.Register(acc, a.onHotkey)
}

func (a *App) onHotkey(pressed bool) {
	a.mu.Lock()
	mode, state := a.cfg.Mode, a.status.State
	a.mu.Unlock()
	slog.Info("hotkey", "pressed", pressed, "state", state, "mode", mode)
	switch {
	case pressed && (state == StateIdle || state == StateError):
		a.Start()
	case pressed && mode == settings.ModeToggle && state == StateRecording:
		a.Stop()
	case !pressed && mode == settings.ModePushToTalk:
		a.Stop()
	}
}

func (a *App) setStatus(s Status) {
	a.mu.Lock()
	a.status = s
	a.mu.Unlock()
	a.wails.Event.Emit(EvState, s)
	a.rebuildMenu()
}

func (a *App) fail(err error) {
	slog.Warn("dictation error", "err", err)
	a.mu.Lock()
	lang := a.status.Language
	if a.errTimer != nil {
		a.errTimer.Stop()
	}
	a.errTimer = time.AfterFunc(3*time.Second, func() {
		a.mu.Lock()
		isErr := a.status.State == StateError
		a.mu.Unlock()
		if isErr {
			a.setStatus(Status{State: StateIdle, Language: lang})
			a.hideOverlay()
		}
	})
	a.mu.Unlock()
	a.setStatus(Status{State: StateError, Error: err.Error(), Language: lang})
	a.showOverlay(true)
}

// ensureEngine loads the selected model if it isn't loaded already.
func (a *App) ensureEngine() error {
	a.mu.Lock()
	id, wantPunct := a.cfg.ModelID, a.cfg.Punctuation
	m, _ := models.Find(id)
	// Apple and Parakeet already punctuate and capitalize.
	needPunct := wantPunct && m.Engine == models.EngineZipformer && m.Language == "en" && a.models.Installed(models.PunctID)
	if a.engine != nil && a.engineID == id && (a.punct != nil || !needPunct) {
		a.mu.Unlock()
		return nil
	}
	if a.status.State != StateIdle && a.status.State != StateError {
		a.mu.Unlock()
		return errors.New("busy")
	}
	a.status.State = StateLoading
	old, oldPunct := a.engine, a.punct
	a.engine, a.punct, a.engineID = nil, nil, ""
	a.mu.Unlock()
	a.wails.Event.Emit(EvState, Status{State: StateLoading})

	if old != nil {
		old.Close()
	}
	if oldPunct != nil {
		oldPunct.Close()
	}
	if _, ok := models.Find(id); !ok || !a.models.Installed(id) {
		a.setStatus(Status{State: StateIdle})
		return fmt.Errorf("speech model %q isn't downloaded yet; open Sayki Settings → Models", id)
	}
	t0 := time.Now()
	var eng transcription.Engine
	var err error
	switch m.Engine {
	case models.EngineApple:
		eng, err = transcription.NewApple()
	case models.EngineParakeet:
		eng, err = transcription.NewParakeet(a.models.Dir(id))
	default:
		eng, err = transcription.NewSherpaTransducer(a.models.Dir(id))
	}
	if err != nil {
		a.setStatus(Status{State: StateIdle})
		return err
	}
	var punct transcription.Punctuator
	if needPunct {
		if punct, err = transcription.NewSherpaPunctuator(a.models.Dir(models.PunctID)); err != nil {
			slog.Warn("punctuation model failed to load; continuing without it", "err", err)
		}
	}
	slog.Info("model loaded", "id", id, "punct", punct != nil, "took", time.Since(t0))
	a.mu.Lock()
	a.engine, a.punct, a.engineID = eng, punct, id
	a.mu.Unlock()
	a.setStatus(Status{State: StateIdle, Language: m.Language})
	return nil
}

// Start begins a dictation session (no-op unless idle).
func (a *App) Start() {
	switch permissions.Check().Microphone {
	case "undetermined":
		permissions.RequestMicrophone()
		a.fail(errors.New("Allow microphone access, then press the shortcut again"))
		return
	case "denied":
		a.fail(errors.New("Microphone access is off — enable Sayki in System Settings → Privacy → Microphone"))
		return
	}
	target := injection.CaptureTarget() // before anything of ours could take focus
	if err := a.ensureEngine(); err != nil {
		a.fail(err)
		return
	}

	a.mu.Lock()
	if a.status.State != StateIdle && a.status.State != StateError {
		a.mu.Unlock()
		return
	}
	mic, eng, punct := a.cfg.MicrophoneID, a.engine, a.punct
	lang := a.status.Language
	a.mu.Unlock()

	capture, err := a.audio.Start(mic)
	if err != nil {
		a.fail(err)
		return
	}
	s := &session{cap: capture, target: target, started: time.Now(), finished: make(chan struct{})}
	a.mu.Lock()
	a.sess = s
	a.mu.Unlock()
	a.setStatus(Status{State: StateRecording, Language: lang, StartedAt: s.started.UnixMilli()})
	a.wails.Event.Emit(EvTranscript, "")
	a.showOverlay(false)
	a.cue(sound.Start)
	slog.Info("recording started")

	stream := eng.NewStream()
	out := make(chan transcription.Event, 64)
	go transcription.Run(stream, punct, capture.Samples, out)
	go a.watch(s)
	go a.collect(s, stream, out, lang)
}

// watch streams levels to the waveform and stops on device errors.
func (a *App) watch(s *session) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-s.finished:
			return
		case err := <-s.cap.Errors:
			a.Stop()
			a.fail(err)
			return
		case <-t.C:
			a.wails.Event.Emit(EvLevel, s.cap.RMS())
		}
	}
}

func (a *App) collect(s *session, stream transcription.Stream, out <-chan transcription.Event, lang string) {
	defer close(s.finished)
	defer stream.Close()
	var finals []string
	for ev := range out {
		if ev.Final {
			finals = append(finals, ev.Text)
			a.wails.Event.Emit(EvTranscript, transcription.Join(finals))
		} else {
			a.wails.Event.Emit(EvTranscript, strings.TrimSpace(transcription.Join(finals)+" "+ev.Text))
		}
	}
	text := transcription.Join(finals)
	a.mu.Lock()
	a.sess = nil
	method, keep, cancelled := a.cfg.InjectMethod, a.cfg.HistoryEnabled, s.cancel
	a.mu.Unlock()
	if n := s.cap.Dropped.Load(); n > 0 {
		slog.Warn("audio chunks dropped", "count", n)
	}
	// Transcript text is never logged.
	slog.Info("recording finished", "secs", time.Since(s.started).Seconds(), "chars", len(text), "cancelled", cancelled)
	if cancelled || text == "" {
		a.setStatus(Status{State: StateIdle, Language: lang})
		a.hideOverlay()
		return
	}
	// Clicking Done makes the overlay panel the key window; hide it first so the
	// paste goes back to the user's app instead of into the overlay.
	application.InvokeSync(func() { a.overlay.Hide() })
	if err := injection.Insert(s.target, text+" ", method); err != nil {
		a.fail(err)
		return
	}
	if keep {
		if err := a.store.AddHistory(text); err != nil {
			slog.Warn("history", "err", err)
		}
	}
	a.setStatus(Status{State: StateIdle, Language: lang})
	a.hideOverlay()
}

// Stop ends the session; the transcript is flushed and inserted.
func (a *App) Stop() {
	a.mu.Lock()
	s := a.sess
	recording := a.status.State == StateRecording
	if recording {
		a.status.State = StateFinishing
	}
	lang := a.status.Language
	cancelled := s != nil && s.cancel
	a.mu.Unlock()
	if s == nil || !recording {
		return
	}
	if cancelled {
		a.cue(sound.Cancel)
	} else {
		a.cue(sound.Stop)
	}
	a.wails.Event.Emit(EvState, Status{State: StateFinishing, Language: lang})
	s.cap.Stop() // closes Samples → pipeline flushes → collect() inserts
}

// cue plays a feedback sound unless the user turned sounds off.
func (a *App) cue(name string) {
	a.mu.Lock()
	on := a.cfg.Sounds
	a.mu.Unlock()
	if on {
		sound.Play(name)
	}
}

// Cancel ends the session without inserting anything.
func (a *App) Cancel() {
	a.mu.Lock()
	if a.sess != nil {
		a.sess.cancel = true
	}
	a.mu.Unlock()
	a.Stop()
}
