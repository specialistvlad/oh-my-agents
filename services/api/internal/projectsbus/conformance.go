package projectsbus

import "github.com/specialistvlad/oh-my-agents/services/api/internal/projects"

// Announcing does not change what a store is.
var _ projects.Store = (*Store)(nil)
