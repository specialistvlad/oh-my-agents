import { describe, expect, it } from 'vitest';

import { NO_TABS, activeTab, close, entomb, focus, open, rename } from './tabs';
import type { Tabs } from './tabs';

const three = (): Tabs =>
  ['a', 'b', 'c'].reduce(
    (tabs, id) => open(tabs, { id, title: id.toUpperCase() }),
    NO_TABS
  );

describe('open', () => {
  it('adds and focuses', () => {
    const tabs = open(NO_TABS, { id: 'a', title: 'A' });
    expect(tabs.open).toHaveLength(1);
    expect(tabs.active).toBe('a');
  });

  it('focuses instead of duplicating', () => {
    const tabs = open(three(), { id: 'a', title: 'A' });
    expect(tabs.open).toHaveLength(3);
    expect(tabs.active).toBe('a');
  });
});

describe('close', () => {
  // Closing several in a row should walk the strip, not jump to the start.
  it('focuses the neighbour', () => {
    const tabs = close(focus(three(), 'b'), 'b');
    expect(tabs.active).toBe('c');
    expect(tabs.open.map((t) => t.id)).toEqual(['a', 'c']);
  });

  it('falls back to the left at the end of the strip', () => {
    expect(close(focus(three(), 'c'), 'c').active).toBe('b');
  });

  it('leaves the focus alone when closing something else', () => {
    expect(close(focus(three(), 'a'), 'c').active).toBe('a');
  });

  it('ends with nothing focused', () => {
    const emptied = ['a', 'b', 'c'].reduce(close, three());
    expect(emptied).toEqual(NO_TABS);
  });

  it('ignores a tab that is not open', () => {
    const tabs = three();
    expect(close(tabs, 'never-opened')).toBe(tabs);
  });
});

describe('entomb', () => {
  // ADR-0011's central rule: a tab whose object someone else deleted stays,
  // and says so. Vanishing would look like a bug and lose what was on screen.
  it('marks the tab without removing it', () => {
    const tabs = entomb(focus(three(), 'b'), 'b');
    expect(tabs.open).toHaveLength(3);
    expect(tabs.open[1].gone).toBe(true);
  });

  it('does not steal or drop the focus', () => {
    const tabs = entomb(focus(three(), 'b'), 'b');
    expect(tabs.active).toBe('b');
    expect(activeTab(tabs)?.gone).toBe(true);
  });

  it('is idempotent', () => {
    const once = entomb(three(), 'a');
    expect(entomb(once, 'a')).toBe(once);
  });

  it('ignores a tab nobody has open', () => {
    const tabs = three();
    expect(entomb(tabs, 'never-opened')).toBe(tabs);
  });

  it('is undone by opening the thing again', () => {
    const back = open(entomb(three(), 'a'), { id: 'a', title: 'A' });
    expect(back.open[0].gone).toBeUndefined();
  });
});

describe('rename', () => {
  it('keeps an open tab current', () => {
    expect(rename(three(), 'a', 'Renamed').open[0].title).toBe('Renamed');
  });

  it('changes nothing when the title already matches', () => {
    const tabs = three();
    expect(rename(tabs, 'a', 'A')).toBe(tabs);
  });
});
