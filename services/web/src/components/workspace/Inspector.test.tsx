// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Item, Schema } from '@/tracker/types';

import { Inspector } from './Inspector';

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
      ],
      transitions: [{ from: 'todo-0001', to: 'doing-0001' }],
    },
  ],
};

const item: Item = {
  id: 'i1',
  type: 'task-0001',
  title: 'Ship it',
  body: '',
  status: 'todo-0001',
  parent: null,
  fields: {},
  rank: 'n',
  version: 2,
  created_at: '',
  updated_at: '',
};

describe('Inspector', () => {
  it('describes what is selected', () => {
    render(
      <Inspector
        item={item}
        missing={false}
        schema={schema}
        busy={false}
        onMove={vi.fn()}
      />
    );
    expect(screen.getByText('Ship it')).toBeDefined();
    expect(screen.getByText('To do')).toBeDefined();
    expect(screen.getByText('2')).toBeDefined();
  });

  // Offering a move the workflow refuses is a button that always fails.
  it('offers only declared moves', async () => {
    const onMove = vi.fn();
    render(
      <Inspector
        item={item}
        missing={false}
        schema={schema}
        busy={false}
        onMove={onMove}
      />
    );
    await userEvent.click(screen.getByRole('button', { name: 'Doing' }));
    expect(onMove).toHaveBeenCalledWith(item, 'doing-0001');
    expect(screen.queryByRole('button', { name: 'To do' })).toBeNull();
  });

  // A selection whose object was deleted says so rather than emptying, for the
  // same reason a tab does.
  it('reports a selection that has been deleted', () => {
    render(
      <Inspector
        item={null}
        missing
        schema={schema}
        busy={false}
        onMove={vi.fn()}
      />
    );
    expect(screen.getByText(/deleted/i)).toBeDefined();
  });

  it('invites a selection when there is none', () => {
    render(
      <Inspector
        item={null}
        missing={false}
        schema={schema}
        busy={false}
        onMove={vi.fn()}
      />
    );
    expect(screen.getByText(/Selecting does not open it/i)).toBeDefined();
  });
});
