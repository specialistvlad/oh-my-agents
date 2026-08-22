import { describe, expect, it } from 'vitest';

import { canDrop, columnsOf, dropAt, isNoop } from './board';
import type { Item, ItemType } from './types';

const task: ItemType = {
  id: 'task-0001',
  name: 'Task',
  initial: 'todo-0001',
  statuses: [
    { id: 'todo-0001', name: 'To do', category: 'backlog' },
    { id: 'doing-0001', name: 'Doing', category: 'active' },
    { id: 'done-0001', name: 'Done', category: 'done' },
  ],
  transitions: [
    { from: 'todo-0001', to: 'doing-0001' },
    { from: 'doing-0001', to: 'done-0001' },
    { from: 'doing-0001', to: 'todo-0001' },
  ],
};

const item = (id: string, status: string, rank: string): Item => ({
  id,
  type: 'task-0001',
  title: id,
  body: '',
  status,
  parent: null,
  fields: {},
  rank,
  version: 1,
  created_at: '',
  updated_at: '',
});

const items = [
  item('a', 'todo-0001', 'n'),
  item('b', 'todo-0001', 't'),
  item('c', 'doing-0001', 'w'),
];

describe('columnsOf', () => {
  it('makes one column per status, in declared order', () => {
    const columns = columnsOf(task, items);
    expect(columns.map((c) => c.status.id)).toEqual([
      'todo-0001',
      'doing-0001',
      'done-0001',
    ]);
  });

  // An empty column is where work goes next; hiding it hides the workflow.
  it('keeps a column nobody is in', () => {
    expect(columnsOf(task, items)[2].items).toEqual([]);
  });

  it('groups cards and preserves the order it was given', () => {
    const columns = columnsOf(task, items);
    expect(columns[0].items.map((i) => i.id)).toEqual(['a', 'b']);
    expect(columns[1].items.map((i) => i.id)).toEqual(['c']);
  });

  it('ignores items of another type', () => {
    const other = { ...item('x', 'todo-0001', 'z'), type: 'bug-0002' };
    expect(
      columnsOf(task, [...items, other])[0].items.map((i) => i.id)
    ).toEqual(['a', 'b']);
  });

  it('has no columns without a type', () => {
    expect(columnsOf(undefined, items)).toEqual([]);
  });
});

describe('canDrop', () => {
  // A dead control teaches people not to trust the interface (ADR-0017).
  it('allows only declared transitions', () => {
    const a = items[0]; // to do
    expect(canDrop(task, a, 'doing-0001')).toBe(true);
    expect(canDrop(task, a, 'done-0001')).toBe(false);
  });

  it('always allows a card back into its own column, for reordering', () => {
    expect(canDrop(task, items[0], 'todo-0001')).toBe(true);
  });

  // Legality is per card, not per type: two cards on one board can have
  // different destinations because they are in different statuses.
  it('depends on the card, not the type', () => {
    expect(canDrop(task, items[0], 'done-0001')).toBe(false);
    expect(canDrop(task, items[2], 'done-0001')).toBe(true);
  });
});

describe('dropAt', () => {
  const todo = columnsOf(task, items)[0];
  const doing = columnsOf(task, items)[1];

  it('lands between two neighbours', () => {
    const three = {
      ...todo,
      items: [items[0], items[1], item('d', 'todo-0001', 'x')],
    };
    expect(dropAt(three, item('e', 'todo-0001', 'z'), 1)).toEqual({
      after: 'a',
      before: 'b',
    });
  });

  it('lands at the start and the end', () => {
    const moving = item('e', 'todo-0001', 'z');
    expect(dropAt(todo, moving, 0)).toEqual({ before: 'a' });
    expect(dropAt(todo, moving, 99)).toEqual({ after: 'b' });
  });

  // Dragging within a column must not treat the card as its own neighbour,
  // which would ask the store to rank something between itself and another.
  it('excludes the card being moved', () => {
    const drop = dropAt(todo, items[0], 1);
    expect(drop.after).not.toBe('a');
    expect(drop.before).not.toBe('a');
    expect(drop).toEqual({ after: 'b' });
  });

  it('names the status only when the column changes', () => {
    expect(dropAt(todo, items[0], 0).status).toBeUndefined();
    expect(dropAt(doing, items[0], 0).status).toBe('doing-0001');
  });

  it('drops into an empty column', () => {
    const done = columnsOf(task, items)[2];
    expect(dropAt(done, items[2], 0)).toEqual({ status: 'done-0001' });
  });
});

describe('isNoop', () => {
  const todo = columnsOf(task, items)[0];

  it('recognises a card dropped where it already is', () => {
    expect(isNoop(dropAt(todo, items[0], 0), todo, items[0])).toBe(true);
  });

  it('does not call a real move a noop', () => {
    expect(isNoop(dropAt(todo, items[0], 1), todo, items[0])).toBe(false);
    const doing = columnsOf(task, items)[1];
    expect(isNoop(dropAt(doing, items[0], 0), doing, items[0])).toBe(false);
  });
});
