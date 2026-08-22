// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  DEFAULT_LAYOUT,
  MIN_WIDTH,
  clampWidth,
  loadLayout,
  saveLayout,
} from './layout';

const WINDOW = 1600;

describe('clampWidth', () => {
  it('keeps a sensible width as it is', () => {
    expect(clampWidth(300, WINDOW)).toBe(300);
  });

  // A column that refuses to shrink and springs back reads as broken.
  it('collapses rather than resisting when dragged small', () => {
    expect(clampWidth(40, WINDOW)).toBe(0);
    expect(clampWidth(0, WINDOW)).toBe(0);
  });

  it('snaps up to the minimum between collapse and usable', () => {
    expect(clampWidth(150, WINDOW)).toBe(MIN_WIDTH);
  });

  // The centre is what the app is for; a side column must not take it over.
  it('never lets one column take the window', () => {
    expect(clampWidth(5000, WINDOW)).toBeLessThanOrEqual(WINDOW * 0.4);
  });

  it('still yields something usable in a narrow window', () => {
    expect(clampWidth(400, 300)).toBe(MIN_WIDTH);
  });
});

describe('remembering the layout', () => {
  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('round trips', () => {
    saveLayout({ left: 200, right: 0, panel: 'activity', view: 'list' });
    expect(loadLayout()).toEqual({
      left: 200,
      right: 0,
      panel: 'activity',
      view: 'list',
    });
  });

  it('falls back to the default when nothing is stored', () => {
    expect(loadLayout()).toEqual(DEFAULT_LAYOUT);
  });

  // A corrupt preference is not worth a blank page.
  it('survives unreadable storage', () => {
    localStorage.setItem('oma.layout', 'not json');
    expect(loadLayout()).toEqual(DEFAULT_LAYOUT);

    localStorage.setItem('oma.layout', '{"left":"wide","panel":"invented"}');
    expect(loadLayout()).toEqual(DEFAULT_LAYOUT);
  });

  it('survives a browser that refuses storage', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('denied');
    });
    expect(loadLayout()).toEqual(DEFAULT_LAYOUT);
    expect(() => saveLayout(DEFAULT_LAYOUT)).not.toThrow();
  });
});
