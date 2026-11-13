// Package settings persists Sayki preferences and optional history in a local SQLite file.
package settings

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	ModePushToTalk = "push"
	ModeToggle     = "toggle"

	InjectPaste = "paste" // clipboard + Cmd+V, clipboard restored afterwards
	InjectType  = "type"  // synthesized unicode keystrokes, clipboard untouched

	OverlayBottom = "bottom"
	OverlayTop    = "top"
	OverlayHidden = "hidden"
)

type Settings struct {
	Shortcut       string `json:"shortcut"` // e.g. "ctrl+option+Space"
	Mode           string `json:"mode"`
	MicrophoneID   string `json:"microphoneId"` // "" = system default
	ModelID        string `json:"modelId"`
	Punctuation    bool   `json:"punctuation"`
	InjectMethod   string `json:"injectMethod"`
	Overlay        string `json:"overlay"`
	LaunchAtLogin  bool   `json:"launchAtLogin"`
	HistoryEnabled bool   `json:"historyEnabled"`
	Sounds         bool   `json:"sounds"` // start/stop/cancel cues
}

func Defaults() Settings {
	return Settings{
		Shortcut:     "ctrl+option+Space",
		Mode:         ModeToggle,
		ModelID:      "en-zipformer",
		Punctuation:  true,
		InjectMethod: InjectPaste,
		Overlay:      OverlayBottom,
		Sounds:       true,
	}
}

type HistoryItem struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store struct{ db *sql.DB }

// DataDir is ~/Library/Application Support/Sayki on macOS (os.UserConfigDir elsewhere).
func DataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Sayki")
	return dir, os.MkdirAll(dir, 0o700)
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(3000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS kv (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS history (id INTEGER PRIMARY KEY AUTOINCREMENT, text TEXT NOT NULL, created_at INTEGER NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Load returns stored settings layered over defaults, so new fields get sane values.
func (s *Store) Load() (Settings, error) {
	st := Defaults()
	var raw string
	err := s.db.QueryRow(`SELECT value FROM kv WHERE key='settings'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal([]byte(raw), &st)
}

func (s *Store) Save(st Settings) error {
	if st.Mode != ModePushToTalk && st.Mode != ModeToggle {
		return fmt.Errorf("invalid mode %q", st.Mode)
	}
	if st.InjectMethod != InjectPaste && st.InjectMethod != InjectType {
		return fmt.Errorf("invalid inject method %q", st.InjectMethod)
	}
	if st.Overlay != OverlayBottom && st.Overlay != OverlayTop && st.Overlay != OverlayHidden {
		return fmt.Errorf("invalid overlay position %q", st.Overlay)
	}
	b, _ := json.Marshal(st)
	_, err := s.db.Exec(`INSERT INTO kv(key,value) VALUES('settings',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(b))
	return err
}

func (s *Store) AddHistory(text string) error {
	_, err := s.db.Exec(`INSERT INTO history(text,created_at) VALUES(?,?)`, text, time.Now().Unix())
	return err
}

func (s *Store) History(limit int) ([]HistoryItem, error) {
	rows, err := s.db.Query(`SELECT id,text,created_at FROM history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []HistoryItem{}
	for rows.Next() {
		var it HistoryItem
		var ts int64
		if err := rows.Scan(&it.ID, &it.Text, &ts); err != nil {
			return nil, err
		}
		it.CreatedAt = time.Unix(ts, 0)
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) ClearHistory() error {
	if _, err := s.db.Exec(`DELETE FROM history`); err != nil {
		return err
	}
	_, err := s.db.Exec(`VACUUM`) // don't leave deleted transcripts in free pages
	return err
}
