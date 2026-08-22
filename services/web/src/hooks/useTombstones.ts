import { useEffect } from 'react';

import type { Item } from '@/tracker/types';

type Shell = {
  entombTab: (id: string) => void;
  renameTab: (id: string, title: string) => void;
};

/**
 * Keeps what is open honest about what still exists.
 *
 * An item can be deleted or renamed by somebody else while it is on screen.
 * A tab whose object is gone is marked rather than removed, and a renamed one
 * follows the new title — the alternative is a tab that quietly describes
 * something that is no longer true (ADR-0011).
 *
 * It waits for `loaded`, because an empty list before the first fetch is not
 * evidence that anything was deleted.
 */
export function useTombstones(
  shell: Shell,
  openIDs: string[],
  items: Item[],
  loaded: boolean
): void {
  const key = openIDs.join(',');
  useEffect(() => {
    if (!loaded) return;
    for (const id of key ? key.split(',') : []) {
      const still = items.find((i) => i.id === id);
      if (still) shell.renameTab(id, still.title);
      else shell.entombTab(id);
    }
  }, [shell, key, items, loaded]);
}
