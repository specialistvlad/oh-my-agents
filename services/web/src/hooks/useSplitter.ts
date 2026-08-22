import { useCallback, useEffect, useRef, useState } from 'react';

import { MIN_WIDTH, clampWidth } from '@/workspace/layout';

/** How far one arrow key press moves a splitter. */
const STEP = 16;

type Options = {
  width: number;
  /** Which side of the window the column is on, so a drag reads correctly. */
  side: 'left' | 'right';
  onResize: (width: number) => void;
};

/**
 * Dragging and keyboard resizing for one splitter.
 *
 * Hand-rolled rather than taken from a package: a splitter is a pointer-move
 * handler and a clamp, and this project has already carried nine unused
 * dependencies once (ADR-0011).
 *
 * Pointer capture is what makes a drag survive the cursor leaving the handle —
 * without it, moving faster than React re-renders drops the drag.
 */
export function useSplitter({ width, side, onResize }: Options) {
  const [dragging, setDragging] = useState(false);
  const latest = useRef(width);
  latest.current = width;

  const onPointerDown = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId);
    setDragging(true);
  }, []);

  const onPointerMove = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!dragging) return;
      const from = side === 'left' ? e.clientX : window.innerWidth - e.clientX;
      onResize(clampWidth(from, window.innerWidth));
    },
    [dragging, side, onResize]
  );

  const onPointerUp = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    e.currentTarget.releasePointerCapture(e.pointerId);
    setDragging(false);
  }, []);

  // Arrow keys move it too. Skipping this is what puts the column widths out
  // of reach of anyone not using a mouse.
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      const grow = side === 'left' ? 'ArrowRight' : 'ArrowLeft';
      const shrink = side === 'left' ? 'ArrowLeft' : 'ArrowRight';
      let next: number | null = null;
      if (e.key === grow) next = (latest.current || MIN_WIDTH) + STEP;
      if (e.key === shrink) next = latest.current - STEP;
      if (e.key === 'Home') next = 0;
      if (e.key === 'End') next = window.innerWidth;
      if (next === null) return;
      e.preventDefault();
      onResize(clampWidth(next, window.innerWidth));
    },
    [side, onResize]
  );

  // While dragging, the cursor belongs to the splitter wherever it goes, and
  // text selection would otherwise highlight the whole page.
  useEffect(() => {
    if (!dragging) return;
    const style = document.body.style;
    const had = { cursor: style.cursor, select: style.userSelect };
    style.cursor = 'col-resize';
    style.userSelect = 'none';
    return () => {
      style.cursor = had.cursor;
      style.userSelect = had.select;
    };
  }, [dragging]);

  return { dragging, onPointerDown, onPointerMove, onPointerUp, onKeyDown };
}
