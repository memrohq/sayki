// Command sayki is the menu-bar voice typing app.
package main

import (
	_ "embed"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/memrohq/sayki/frontend"
	"github.com/memrohq/sayki/internal/app"
	"github.com/memrohq/sayki/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed tray.png
var trayIcon []byte

func main() {
	dir, err := settings.DataDir()
	if err != nil {
		log.Fatal(err)
	}
	logFile, err := os.OpenFile(filepath.Join(dir, "sayki.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatal(err)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stderr, logFile), nil)))

	core, err := app.New(dir)
	if err != nil {
		slog.Error("startup", "err", err)
		os.Exit(1)
	}
	w := application.New(application.Options{
		Name:        "Sayki",
		Description: "Private, offline voice typing",
		Services:    []application.Service{application.NewService(core)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(frontend.Dist)},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory, // menu bar only, no Dock icon
		},
		SingleInstance: &application.SingleInstanceOptions{UniqueID: "co.memro.sayki"},
	})
	core.Attach(w, trayIcon)
	if err := w.Run(); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}
