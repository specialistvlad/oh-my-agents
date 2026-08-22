import { useCallback, useState } from 'react';

import type { Item } from '@/tracker/types';

/** A drag in progress, and where it is hovering. */
export type Drag = {
  item: Item | null;
  over: { status: string; at: number } | null;
  start: (item: Item) => void;
  end: () => void;
  hover: (status: string, at: number) => void;
  leave: (status: string) => void;
};

/**
 * Which card is being dragged, and where it is hovering.
 *
 * Kept in one place because a drag spans two components — the card that starts
 * it and the column it ends over — and neither owns the other.
 */
export function useBoardDrag(): Drag {
  const [item, setItem] = useState<Item | null>(null);
  const [over, setOver] = useState<{ status: string; at: number } | null>(null);

  const end = useCallback(() => {
    setItem(null);
    setOver(null);
  }, []);

  return {
    item,
    over,
    start: useCallback((dragged: Item) => setItem(dragged), []),
    end,
    hover: useCallback(
      (status: string, at: number) => setOver({ status, at }),
      []
    ),
    // Leaving one column while entering another fires in that order, so a
    // leave must not clear a hover the next column has already set.
    leave: useCallback(
      (status: string) =>
        setOver((held) => (held?.status === status ? null : held)),
      []
    ),
  };
}
