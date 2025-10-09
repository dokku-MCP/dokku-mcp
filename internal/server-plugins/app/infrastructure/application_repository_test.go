package infrastructure

import (
	"testing"

	app "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/domain"
)

func TestDetermineStateFromInfo(t *testing.T) {
	repo := &DokkuApplicationRepository{}

	t.Run("fresh app reports exists", func(t *testing.T) {
		info := map[string]string{
			"App locked":        "true",
			"Deployed":          "false",
			"App deploy source": "",
			"Processes":         "0",
		}

		state := repo.determineStateFromInfo(info)
		if state != app.StateExists {
			t.Fatalf("expected state exists, got %s", state)
		}
	})

	t.Run("locked app with deploy source reports error", func(t *testing.T) {
		info := map[string]string{
			"App locked":        "true",
			"Deployed":          "false",
			"App deploy source": "github",
			"Processes":         "0",
		}

		state := repo.determineStateFromInfo(info)
		if state != app.StateError {
			t.Fatalf("expected state error, got %s", state)
		}
	})
}
