/** The panels the icon rail can show. Objects is first and the default. */
export type PanelID = 'objects' | 'activity';

/** What the centre shows when nothing is open. Local, like the widths. */
export type View = 'board' | 'list';

export type Layout = {
  /** Column widths in pixels. Zero means collapsed. */
  left: number;
  right: number;
  panel: PanelID;
  view: View;
};

export const DEFAULT_LAYOUT: Layout = {
  left: 260,
  right: 300,
  panel: 'objects',
  view: 'board',
};

/** Below this a column is unusable, so dragging past it collapses instead. */
export const MIN_WIDTH = 180;

/** Dragged narrower than this, a column collapses rather than resisting. */
const COLLAPSE_AT = 120;

/** No column may take the whole window; the centre is what the app is for. */
const MAX_FRACTION = 0.4;

/**
 * The width a drag should produce.
 *
 * Collapsing rather than clamping at the minimum is what makes a splitter feel
 * like it does something: a column that refuses to go below 180px and then
 * springs back reads as broken. Zero is recoverable — the rail brings the left
 * back, the inspector toggle the right — so nothing is lost by allowing it.
 */
export function clampWidth(px: number, windowWidth: number): number {
  if (px < COLLAPSE_AT) return 0;
  const max = Math.max(MIN_WIDTH, Math.floor(windowWidth * MAX_FRACTION));
  return Math.min(Math.max(px, MIN_WIDTH), max);
}

const STORED = 'oma.layout';

/**
 * Reads the remembered layout.
 *
 * Per device, not per user (ADR-0011): a laptop and a wide monitor want
 * different widths, so syncing them would make both worse. Anything unreadable
 * falls back to the default rather than failing — a corrupt preference is not
 * worth a blank page.
 */
export function loadLayout(): Layout {
  try {
    const held = localStorage.getItem(STORED);
    if (!held) return DEFAULT_LAYOUT;
    const parsed = JSON.parse(held) as Partial<Layout>;
    return {
      left: typeof parsed.left === 'number' ? parsed.left : DEFAULT_LAYOUT.left,
      right:
        typeof parsed.right === 'number' ? parsed.right : DEFAULT_LAYOUT.right,
      panel: parsed.panel === 'activity' ? 'activity' : 'objects',
      view: parsed.view === 'list' ? 'list' : 'board',
    };
  } catch {
    return DEFAULT_LAYOUT;
  }
}

export function saveLayout(layout: Layout): void {
  try {
    localStorage.setItem(STORED, JSON.stringify(layout));
  } catch {
    // A browser refusing storage is not a reason to stop working.
  }
}
