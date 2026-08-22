package projects

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/ids"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// keyPrefix is where records live in the shared settings store. A project
// record is a JSON document addressed by an id, which is exactly what a
// settings store already is — so there is no second store here, and none of
// its guarantees to keep aligned (ADR-0010).
const keyPrefix = "projects/"

// Registry is the project store.
type Registry struct {
	records   settings.Store
	workspace string
	clock     func() time.Time
	nonce     func() string
}

// Deps are the ambient dependencies. Both clocks default to the real thing.
type Deps struct {
	// Records holds the registry, and is expected to be a store rooted in
	// the shared scope.
	Records settings.Store
	// Workspace is the .oma root, used for the default project root and to
	// refuse a root that would contain it.
	Workspace string
	Clock     func() time.Time
	Nonce     func() string
}

// NewRegistry returns a registry over a records store.
func NewRegistry(d Deps) *Registry {
	if d.Clock == nil {
		d.Clock = func() time.Time { return time.Now().UTC() }
	}
	if d.Nonce == nil {
		d.Nonce = ids.Nonce
	}
	return &Registry{records: d.Records, workspace: d.Workspace, clock: d.Clock, nonce: d.Nonce}
}

// Create implements [Store].
func (r *Registry) Create(ctx context.Context, n New) (Project, error) {
	if err := ValidateName(n.Name); err != nil {
		return Project{}, err
	}
	id := MintID(n.Name, r.nonce())

	root := n.Root
	if root == "" {
		root = filepath.Join(r.workspace, "projects", string(id))
	}
	abs, err := resolveRoot(root, r.workspace)
	if err != nil {
		return Project{}, err
	}
	if err := writeMarker(abs, id); err != nil {
		return Project{}, err
	}
	now := r.clock()
	p := Project{
		ID:        id,
		Name:      strings.TrimSpace(n.Name),
		Root:      abs,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return p, r.put(ctx, p)
}

// Get implements [Store].
func (r *Registry) Get(ctx context.Context, id ID) (Project, error) {
	if err := id.Validate(); err != nil {
		return Project{}, err
	}
	p, err := settings.Read[Project](ctx, r.records, key(id))
	if errors.Is(err, settings.ErrNotFound) {
		return Project{}, notFound(id)
	}
	return p, err
}

// List implements [Store].
func (r *Registry) List(ctx context.Context) ([]Project, error) {
	keys, err := r.records.Keys(ctx)
	if err != nil {
		return nil, err
	}
	var out []Project
	for _, k := range keys {
		if !strings.HasPrefix(string(k), keyPrefix) {
			continue
		}
		p, err := settings.Read[Project](ctx, r.records, k)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Project) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(string(a.ID), string(b.ID))
	})
	return out, nil
}

func key(id ID) settings.Key { return settings.Key(keyPrefix + string(id)) }

func (r *Registry) put(ctx context.Context, p Project) error {
	return settings.Write(ctx, r.records, key(p.ID), p)
}
