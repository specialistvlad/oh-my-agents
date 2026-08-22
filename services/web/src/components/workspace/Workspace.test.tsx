// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { DEFAULT_LAYOUT } from '@/workspace/layout';

import { Workspace } from './Workspace';

const shell = (layout = DEFAULT_LAYOUT, handlers = {}) => (
  <Workspace
    layout={layout}
    left={<div>objects panel</div>}
    centre={<div>centre</div>}
    right={<div>inspector</div>}
    onPanel={vi.fn()}
    onResizeLeft={vi.fn()}
    onResizeRight={vi.fn()}
    {...handlers}
  />
);

// Cleanup is not automatic without vitest globals, and renders otherwise
// accumulate in the document until every query finds several matches.
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('Workspace', () => {
  it('shows three columns behind the rail', () => {
    render(shell());
    expect(screen.getByText('objects panel')).toBeDefined();
    expect(screen.getByText('centre')).toBeDefined();
    expect(screen.getByText('inspector')).toBeDefined();
    expect(screen.getByRole('navigation', { name: 'Panels' })).toBeDefined();
  });

  // A column at zero width still takes its borders and its focus order, and a
  // splitter beside nothing resizes air.
  it('does not render a collapsed column at all', () => {
    render(shell({ ...DEFAULT_LAYOUT, left: 0 }));
    expect(screen.queryByText('objects panel')).toBeNull();
    expect(screen.queryByRole('separator', { name: /objects/i })).toBeNull();
    expect(screen.getByText('centre')).toBeDefined();
  });

  it('keeps the rail so a collapsed column can be recovered', () => {
    render(shell({ ...DEFAULT_LAYOUT, left: 0 }));
    expect(screen.getByRole('button', { name: 'Objects' })).toBeDefined();
  });

  // Skipping keyboard resizing puts the widths out of reach of anyone not
  // using a mouse (ADR-0011).
  it('resizes with the keyboard', async () => {
    const onResizeLeft = vi.fn();
    render(shell(DEFAULT_LAYOUT, { onResizeLeft }));

    const handle = screen.getByRole('separator', { name: /objects/i });
    handle.focus();
    await userEvent.keyboard('{ArrowRight}');
    expect(onResizeLeft).toHaveBeenCalledWith(DEFAULT_LAYOUT.left + 16);

    await userEvent.keyboard('{Home}');
    expect(onResizeLeft).toHaveBeenLastCalledWith(0);
  });

  it('reports its width on the separator', () => {
    render(shell());
    const handle = screen.getByRole('separator', { name: /objects/i });
    expect(handle.getAttribute('aria-valuenow')).toBe(
      String(DEFAULT_LAYOUT.left)
    );
  });
});
