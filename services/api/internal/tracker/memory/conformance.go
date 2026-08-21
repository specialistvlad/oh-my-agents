package memory

import "github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"

// Store implements every port. The assertion is here rather than in a test so
// that a missing method breaks the build, not a test run.
var _ tracker.Store = (*Store)(nil)
