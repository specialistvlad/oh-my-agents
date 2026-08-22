import { describe, expect, it } from 'vitest';

import { ITEM_DELETED, canMove, itemOf, movesFrom, statusOf } from './types';
import type { ItemType } from './types';

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
  ],
};

describe('itemOf', () => {
  it('names the item an event concerns', () => {
    expect(itemOf('item_created', { item: 'a1' })).toBe('a1');
    expect(itemOf('status_changed', { item: 'a1' })).toBe('a1');
    expect(itemOf(ITEM_DELETED, { item: 'a1' })).toBe('a1');
  });

  // Comments and links reach the same room; only item events move the list.
  it('ignores events about something else', () => {
    expect(itemOf('comment_added', { item: 'a1' })).toBeNull();
    expect(itemOf('project.changed', { id: 'p1' })).toBeNull();
  });

  it('survives a payload it cannot read', () => {
    expect(itemOf('item_created', undefined)).toBeNull();
    expect(itemOf('item_created', {})).toBeNull();
  });
});

describe('transitions', () => {
  // Offering a move the workflow refuses produces a button that always
  // fails, so the UI asks the type rather than showing every status.
  it('offers only declared moves', () => {
    expect(movesFrom(task, 'todo-0001').map((s) => s.id)).toEqual([
      'doing-0001',
    ]);
    expect(movesFrom(task, 'done-0001')).toEqual([]);
  });

  it('never offers the status it is already in', () => {
    expect(movesFrom(task, 'doing-0001').map((s) => s.id)).not.toContain(
      'doing-0001'
    );
  });

  it('knows what is allowed', () => {
    expect(canMove(task, 'todo-0001', 'doing-0001')).toBe(true);
    expect(canMove(task, 'todo-0001', 'done-0001')).toBe(false);
  });

  it('reads a status by id', () => {
    expect(statusOf(task, 'doing-0001')?.name).toBe('Doing');
    expect(statusOf(task, 'gone')).toBeUndefined();
    expect(statusOf(undefined, 'doing-0001')).toBeUndefined();
  });
});
