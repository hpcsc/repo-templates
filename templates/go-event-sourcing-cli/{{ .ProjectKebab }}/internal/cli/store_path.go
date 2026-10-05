package cli

import (
	"os"
	"path/filepath"
)

func defaultStorePath(app string) string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, app, "events.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return app + ".db"
	}
	return filepath.Join(home, ".local", "share", app, "events.db")
}
