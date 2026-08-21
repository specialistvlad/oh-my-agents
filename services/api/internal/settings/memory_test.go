package settings_test

import (
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings/settingstest"
)

func TestMemoryConformance(t *testing.T) {
	settingstest.Run(t, func(_ *testing.T) settings.Store {
		return settings.NewMemory()
	})
}
