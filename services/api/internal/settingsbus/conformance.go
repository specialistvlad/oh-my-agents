package settingsbus

import "github.com/specialistvlad/oh-my-agents/services/api/internal/settings"

// Announcing does not change what a store is. The assertion is here so that a
// method drifting out of shape breaks the build rather than a call site.
var _ settings.Store = (*Store)(nil)
