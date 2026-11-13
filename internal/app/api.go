package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/memrohq/sayki/internal/audio"
	"github.com/memrohq/sayki/internal/models"
	"github.com/memrohq/sayki/internal/permissions"
	"github.com/memrohq/sayki/internal/settings"
	"github.com/memrohq/sayki/internal/shortcuts"
)

// Methods below are called from the frontend via Wails bindings.

func (a *App) GetSettings() settings.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

func (a *App) GetStatus() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

func (a *App) SaveSettings(s settings.Settings) error {
	if _, err := shortcuts.Parse(s.Shortcut); err != nil {
		return err
	}
	if _, ok := models.Find(s.ModelID); !ok {
		return errors.New("unknown model")
	}
	a.mu.Lock()
	old := a.cfg
	a.mu.Unlock()

	if s.Shortcut != old.Shortcut {
		if err := a.registerShortcut(s.Shortcut); err != nil {
			a.registerShortcut(old.Shortcut) // keep a working shortcut
			return err
		}
	}
	if s.LaunchAtLogin != old.LaunchAtLogin {
		var err error
		if s.LaunchAtLogin {
			err = a.wails.Autostart.Enable()
		} else {
			err = a.wails.Autostart.Disable()
		}
		if err != nil {
			return errors.New("launch at login needs Sayki to run from its .app bundle: " + err.Error())
		}
	}
	if err := a.store.Save(s); err != nil {
		return err
	}
	if !s.HistoryEnabled && old.HistoryEnabled {
		if err := a.store.ClearHistory(); err != nil {
			slog.Warn("clear history", "err", err)
		}
	}
	a.mu.Lock()
	a.cfg = s
	if s.Overlay != old.Overlay {
		a.placed = false // snap back to the newly chosen top/bottom spot
	}
	a.mu.Unlock()
	a.wails.Event.Emit(EvSettings, s)
	a.rebuildMenu()
	if s.ModelID != old.ModelID || s.Punctuation != old.Punctuation {
		go a.ensureEngine()
	}
	return nil
}

func (a *App) update(fn func(*settings.Settings)) {
	s := a.GetSettings()
	fn(&s)
	if err := a.SaveSettings(s); err != nil {
		a.fail(err)
	}
}

// SuspendShortcut releases the global hotkey while the user records a new one.
func (a *App) SuspendShortcut() { shortcuts.Unregister() }

func (a *App) ResumeShortcut() error { return a.registerShortcut(a.GetSettings().Shortcut) }

func (a *App) ListMicrophones() ([]audio.Device, error) { return a.audio.Microphones() }

func (a *App) ListModels() []ModelInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ModelInfo, 0, len(models.Catalog))
	for _, m := range models.Catalog {
		_, dl := a.downloads[m.ID]
		out = append(out, ModelInfo{Model: m, Installed: a.models.Installed(m.ID), Downloading: dl})
	}
	return out
}

// DownloadModel starts a download in the background; progress arrives as EvDownload.
func (a *App) DownloadModel(id string) error {
	if _, ok := models.Find(id); !ok {
		return errors.New("unknown model")
	}
	a.mu.Lock()
	if _, busy := a.downloads[id]; busy {
		a.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(a.shutdown)
	a.downloads[id] = cancel
	a.mu.Unlock()

	go func() {
		slog.Info("model download started", "id", id)
		err := a.models.Download(ctx, id, func(done, total int64) {
			a.wails.Event.Emit(EvDownload, DownloadProgress{ID: id, Done: done, Total: total})
		})
		a.mu.Lock()
		delete(a.downloads, id)
		a.mu.Unlock()
		p := DownloadProgress{ID: id, Finished: true}
		if err != nil {
			p.Error = err.Error()
			if errors.Is(err, context.Canceled) {
				p.Error = "cancelled"
			}
		}
		slog.Info("model download finished", "id", id, "err", err)
		a.wails.Event.Emit(EvDownload, p)
		a.rebuildMenu()
		if err == nil {
			go a.ensureEngine()
		}
	}()
	return nil
}

func (a *App) CancelDownload(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if c, ok := a.downloads[id]; ok {
		c()
	}
}

func (a *App) DeleteModel(id string) error {
	a.mu.Lock()
	inUse := a.engineID == id && a.status.State != StateIdle
	if a.engineID == id && !inUse {
		a.engine.Close()
		a.engine, a.engineID = nil, ""
	}
	if id == models.PunctID && a.punct != nil && !inUse {
		a.punct.Close()
		a.punct = nil
	}
	a.mu.Unlock()
	if inUse {
		return errors.New("model is in use; stop dictation first")
	}
	err := a.models.Delete(id)
	a.rebuildMenu()
	return err
}

func (a *App) Permissions() permissions.Status { return permissions.Check() }

func (a *App) RequestPermission(kind string) {
	if kind == "microphone" {
		permissions.RequestMicrophone()
	} else {
		permissions.RequestAccessibility()
	}
}

func (a *App) OpenPermissionSettings(kind string) error { return permissions.OpenSettings(kind) }

func (a *App) History() ([]settings.HistoryItem, error) { return a.store.History(200) }

func (a *App) ClearHistory() error { return a.store.ClearHistory() }

func (a *App) ToggleRecording() {
	switch a.GetStatus().State {
	case StateRecording:
		a.Stop()
	case StateIdle, StateError:
		go a.Start()
	}
}

// ShortcutLabel renders "ctrl+option+Space" as "⌃⌥Space".
func ShortcutLabel(s string) string {
	if l, ok := shortcuts.SoloKeys[s]; ok {
		return l
	}
	r := strings.NewReplacer("ctrl+", "⌃", "option+", "⌥", "shift+", "⇧", "cmd+", "⌘", "Key", "", "Digit", "")
	return r.Replace(s)
}
