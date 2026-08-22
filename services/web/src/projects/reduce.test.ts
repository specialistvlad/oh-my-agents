import { describe, expect, it } from 'vitest';

import { reduce, sort } from './reduce';
import { PROJECT_CHANGED, PROJECT_CREATED, PROJECT_REMOVED } from './types';
import type { Project } from './types';

const project = (id: string, name: string): Project => ({
  id,
  name,
  root: `/w/${id}`,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
});

describe('reduce', () => {
  it('adds a created project in order', () => {
    const after = reduce(
      [project('z-1', 'Zebra')],
      PROJECT_CREATED,
      project('a-1', 'Apple')
    );
    expect(after.map((p) => p.name)).toEqual(['Apple', 'Zebra']);
  });

  it('replaces a changed project and re-sorts', () => {
    const before = [project('a-1', 'Apple'), project('z-1', 'Zebra')];
    const after = reduce(before, PROJECT_CHANGED, project('a-1', 'Zucchini'));
    expect(after.map((p) => p.name)).toEqual(['Zebra', 'Zucchini']);
    expect(after).toHaveLength(2);
  });

  it('removes by id', () => {
    const before = [project('a-1', 'Apple'), project('z-1', 'Zebra')];
    expect(
      reduce(before, PROJECT_REMOVED, { id: 'a-1' }).map((p) => p.id)
    ).toEqual(['z-1']);
  });

  // A fetch and an event will describe the same state; applying both must not
  // duplicate anything.
  it('is idempotent', () => {
    const created = project('a-1', 'Apple');
    const once = reduce([], PROJECT_CREATED, created);
    const twice = reduce(once, PROJECT_CREATED, created);
    expect(twice).toHaveLength(1);
  });

  it('ignores events it cannot use', () => {
    const before = [project('a-1', 'Apple')];
    expect(reduce(before, PROJECT_CREATED, undefined)).toBe(before);
    expect(reduce(before, PROJECT_REMOVED, {})).toBe(before);
    expect(reduce(before, 'something.else', project('b-1', 'B'))).toBe(before);
  });

  it('does not mutate the list it was given', () => {
    const before = [project('a-1', 'Apple')];
    reduce(before, PROJECT_CREATED, project('b-1', 'Banana'));
    expect(before).toHaveLength(1);
  });
});

describe('sort', () => {
  it('orders by name then id, case-insensitively', () => {
    const all = [
      project('b-2', 'beta'),
      project('b-1', 'Beta'),
      project('a-1', 'alpha'),
    ];
    expect(sort(all).map((p) => p.id)).toEqual(['a-1', 'b-1', 'b-2']);
  });
});
