// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Drag } from '@/hooks/useBoardDrag';
import type { Item, Schema } from '@/tracker/types';

import { Board } from './Board';

afterEach(cleanup);

const schema: Schema = {
  types: [
    {
      id: 'task-0001',
      name: 'Task',
      initial: 'todo-0001',
      statuses: [
        { id: 'todo-0001', name: 'To do', category: 'backlog' },
        { id: 'doing-0001', name: 'Doing', category: 'active' },
        { id: 'done-0001', name: 'Done', category: 'done' },
      ],
      transitions: [{ from: 'todo-0001', to: 'doing-0001' }],
    },
  ],
};

const card = (id: string, status: string, rank: string): Item => ({
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
  card('alpha', 'todo-0001', 'n'),
  card('beta', 'doing-0001', 't'),
];

const idle: Drag = {
  item: null,
  over: null,
  start: vi.fn(),
  end: vi.fn(),
  hover: vi.fn(),
  leave: vi.fn(),
};

const board = (drag: Drag = idle) => (
  <Board
    schema={schema}
    items={items}
    loaded
    selected={null}
    drag={drag}
    onSelect={vi.fn()}
    onOpen={vi.fn()}
    onDrop={vi.fn()}
  />
);

describe('Board', () => {
  it('shows a column per status, including empty ones', () => {
    render(board());
    for (const name of ['To do', 'Doing', 'Done']) {
      expect(screen.getByRole('region', { name })).toBeDefined();
    }
  });

  it('puts each card in its own column', () => {
    render(board());
    const todo = screen.getByRole('region', { name: 'To do' });
    expect(todo.textContent).toContain('alpha');
    expect(todo.textContent).not.toContain('beta');
  });

  // ADR-0017's central claim. A column the card cannot reach must not accept
  // a drop, because a control that always fails teaches people not to trust
  // the interface.
  it('refuses a drop the workflow does not allow', () => {
    const onDrop = vi.fn();
    const dragging: Drag = {
      ...idle,
      item: items[0], // in "To do", which can only reach "Doing"
      over: { status: 'done-0001', at: 0 },
    };
    render(
      <Board
        schema={schema}
        items={items}
        loaded
        selected={null}
        drag={dragging}
        onSelect={vi.fn()}
        onOpen={vi.fn()}
        onDrop={onDrop}
      />
    );
    const done = screen.getByRole('region', { name: 'Done' });
    done.dispatchEvent(new Event('drop', { bubbles: true }));
    expect(onDrop).not.toHaveBeenCalled();
  });

  // Legality is per card: two cards on one board have different destinations
  // because they are in different statuses.
  it('marks reachable and unreachable columns differently', () => {
    render(board({ ...idle, item: items[0] }));
    const reachable = screen.getByRole('region', { name: 'Doing' });
    const barred = screen.getByRole('region', { name: 'Done' });
    expect(barred.className).toContain('opacity-40');
    expect(reachable.className).not.toContain('opacity-40');
  });

  it('says so when the project has no types', () => {
    render(
      <Board
        schema={{ types: [] }}
        items={[]}
        loaded
        selected={null}
        drag={idle}
        onSelect={vi.fn()}
        onOpen={vi.fn()}
        onDrop={vi.fn()}
      />
    );
    expect(screen.getByText(/no types/i)).toBeDefined();
  });
});
