// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TabStrip } from './TabStrip';

afterEach(cleanup);

const tabs = [
  { id: 'a', title: 'Alpha' },
  { id: 'b', title: 'Beta', gone: true },
];

describe('TabStrip', () => {
  it('shows what is open and which is focused', () => {
    render(
      <TabStrip tabs={tabs} active="a" onFocus={vi.fn()} onClose={vi.fn()} />
    );
    expect(
      screen.getByRole('tab', { name: 'Alpha' }).getAttribute('aria-selected')
    ).toBe('true');
    expect(
      screen.getByRole('tab', { name: 'Beta' }).getAttribute('aria-selected')
    ).toBe('false');
  });

  // ADR-0011's central rule. A tab that vanishes mid-sentence looks like a bug
  // in the application and slides every neighbour under the cursor.
  it('keeps a tab whose object is gone, and says so', () => {
    render(
      <TabStrip tabs={tabs} active="a" onFocus={vi.fn()} onClose={vi.fn()} />
    );
    const gone = screen.getByRole('tab', { name: 'Beta' });
    expect(gone).toBeDefined();
    expect(gone.className).toContain('line-through');
    expect(gone.getAttribute('title')).toContain('deleted');
  });

  it('closes only when asked', async () => {
    const onClose = vi.fn();
    render(
      <TabStrip tabs={tabs} active="a" onFocus={vi.fn()} onClose={onClose} />
    );
    await userEvent.click(screen.getByRole('button', { name: 'Close Beta' }));
    expect(onClose).toHaveBeenCalledWith('b');
  });

  it('renders nothing when nothing is open', () => {
    const { container } = render(
      <TabStrip tabs={[]} active={null} onFocus={vi.fn()} onClose={vi.fn()} />
    );
    expect(container.firstChild).toBeNull();
  });
});
