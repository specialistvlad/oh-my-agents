import { useCallback, useEffect, useState } from 'react';

import { DEFAULT_LAYOUT, loadLayout, saveLayout } from '@/workspace/layout';
import type { Layout, PanelID } from '@/workspace/layout';
import { NO_TABS, close, entomb, focus, open, rename } from '@/workspace/tabs';
import type { Tab, Tabs } from '@/workspace/tabs';

/**
 * The shell's own state: column widths, which panel the rail shows, what is
 * open in the centre, and what is selected.
 *
 * Selection and the active tab are separate, which is ADR-0011's deliberate
 * cost: the inspector follows selection, so you can look at one thing while
 * working on another. Keeping them in one place is what lets the UI show both
 * distinctly — a person who cannot tell which the inspector describes will
 * stop trusting it.
 */
export function useWorkspace() {
  const [layout, setLayout] = useState<Layout>(DEFAULT_LAYOUT);
  const [tabs, setTabs] = useState<Tabs>(NO_TABS);
  const [selected, setSelected] = useState<string | null>(null);
  const [draft, setDraft] = useState('');

  // Read after mount, not during render: the server has no localStorage, and
  // a value read during render would differ between the two.
  useEffect(() => setLayout(loadLayout()), []);

  const change = useCallback((next: Layout) => {
    setLayout(next);
    saveLayout(next);
  }, []);

  return {
    layout,
    tabs,
    selected,
    draft,
    setDraft,

    resizeLeft: useCallback(
      (left: number) => change({ ...layout, left }),
      [change, layout]
    ),
    resizeRight: useCallback(
      (right: number) => change({ ...layout, right }),
      [change, layout]
    ),

    // Choosing the panel already showing collapses it, the way an IDE
    // sidebar behaves, and choosing another opens it again.
    selectPanel: useCallback(
      (panel: PanelID) => {
        const showing = layout.left > 0 && layout.panel === panel;
        change({
          ...layout,
          panel,
          left: showing ? 0 : layout.left || DEFAULT_LAYOUT.left,
        });
      },
      [change, layout]
    ),
    toggleInspector: useCallback(
      () =>
        change({
          ...layout,
          right: layout.right > 0 ? 0 : DEFAULT_LAYOUT.right,
        }),
      [change, layout]
    ),

    /** Selecting shows a thing in the inspector without opening it. */
    select: setSelected,
    openTab: useCallback((tab: Tab) => setTabs((t) => open(t, tab)), []),
    focusTab: useCallback((id: string) => setTabs((t) => focus(t, id)), []),
    closeTab: useCallback((id: string) => setTabs((t) => close(t, id)), []),
    renameTab: useCallback(
      (id: string, title: string) => setTabs((t) => rename(t, id, title)),
      []
    ),
    /** The object is gone; its tab stays and says so. */
    entombTab: useCallback((id: string) => setTabs((t) => entomb(t, id)), []),
  };
}
