package scopes

import (
	"context"
	"fmt"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Identifiers for the starter type. Fixed rather than minted, because they
// are the same type in every project and a client that knows how to render a
// task should not have to look up a different id per project.
const (
	StarterType   tracker.TypeID   = "task-0001"
	StatusTodo    tracker.StatusID = "todo-0001"
	StatusDoing   tracker.StatusID = "doing-0001"
	StatusDone    tracker.StatusID = "done-0001"
	StatusDropped tracker.StatusID = "dropped-0001"
)

// seed gives a brand new tracker one type to work with.
//
// Without it a project has a tracker that can hold nothing: an item needs a
// type, and nothing yet authors one. This is a starting point rather than a
// fixture — a project is free to add types, rename this one, or delete it once
// something else exists.
//
// It runs only when the schema is empty, so it never overwrites anything and
// never comes back after being deleted.
func seed(ctx context.Context, s tracker.Store) error {
	schema, err := s.Schema(ctx)
	if err != nil {
		return err
	}
	if len(schema.Types) > 0 {
		return nil
	}
	if err := s.PutItemType(ctx, StarterTaskType()); err != nil {
		return fmt.Errorf("scopes: seeding the starter type: %w", err)
	}
	return nil
}

// StarterTaskType is the type a new project begins with: a plain task with a
// linear workflow and a way to abandon it.
//
// It declares no custom fields. An item already has a title and a body, which
// is enough to be useful, and every field added here would be one every
// project had to live with.
func StarterTaskType() tracker.ItemType {
	return tracker.ItemType{
		ID:   StarterType,
		Name: "Task",
		Statuses: []tracker.Status{
			{ID: StatusTodo, Name: "To do", Category: tracker.CategoryBacklog},
			{ID: StatusDoing, Name: "Doing", Category: tracker.CategoryActive},
			{ID: StatusDone, Name: "Done", Category: tracker.CategoryDone},
			{ID: StatusDropped, Name: "Dropped", Category: tracker.CategoryCanceled},
		},
		Initial: StatusTodo,
		Transitions: []tracker.Transition{
			{From: StatusTodo, To: StatusDoing},
			{From: StatusDoing, To: StatusTodo},
			{From: StatusDoing, To: StatusDone},
			{From: StatusDone, To: StatusDoing},
			{From: StatusTodo, To: StatusDropped},
			{From: StatusDoing, To: StatusDropped},
			{From: StatusDropped, To: StatusTodo},
		},
	}
}
