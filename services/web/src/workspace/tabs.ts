/** One thing open in the centre. */
export type Tab = {
  id: string;
  title: string;
  /** The object behind this tab is gone. The tab stays and says so. */
  gone?: boolean;
};

export type Tabs = {
  open: Tab[];
  /** The focused tab, or null when nothing is open. */
  active: string | null;
};

export const NO_TABS: Tabs = { open: [], active: null };

/** Opens a thing, or focuses it if it is already open. */
export function open(tabs: Tabs, tab: Tab): Tabs {
  const at = tabs.open.findIndex((t) => t.id === tab.id);
  if (at < 0) {
    return { open: [...tabs.open, tab], active: tab.id };
  }
  // Reopening something entombed brings it back, which is what happens when
  // an id is reused or a delete is undone.
  const next = [...tabs.open];
  next[at] = tab;
  return { open: next, active: tab.id };
}

/**
 * Closes a tab, focusing its neighbour.
 *
 * The neighbour rather than the first tab: closing several in a row should
 * walk along the strip, not jump back to the start each time.
 */
export function close(tabs: Tabs, id: string): Tabs {
  const at = tabs.open.findIndex((t) => t.id === id);
  if (at < 0) return tabs;

  const remaining = tabs.open.filter((t) => t.id !== id);
  if (tabs.active !== id) {
    return { open: remaining, active: tabs.active };
  }
  const neighbour = remaining[at] ?? remaining[at - 1] ?? null;
  return { open: remaining, active: neighbour?.id ?? null };
}

/**
 * Marks a tab's object as gone, without removing the tab.
 *
 * This is ADR-0011's central rule. A tab that vanishes mid-sentence looks like
 * a bug in the application, loses whatever was on screen, and slides every
 * neighbouring tab under the cursor — so an object deleted by somebody else
 * leaves a tab that says so and waits to be closed deliberately.
 */
export function entomb(tabs: Tabs, id: string): Tabs {
  const at = tabs.open.findIndex((t) => t.id === id);
  if (at < 0 || tabs.open[at].gone) return tabs;

  const next = [...tabs.open];
  next[at] = { ...next[at], gone: true };
  return { open: next, active: tabs.active };
}

/** Renames an open tab, so a title edited elsewhere is not stale here. */
export function rename(tabs: Tabs, id: string, title: string): Tabs {
  const at = tabs.open.findIndex((t) => t.id === id);
  if (at < 0 || tabs.open[at].title === title) return tabs;

  const next = [...tabs.open];
  next[at] = { ...next[at], title };
  return { open: next, active: tabs.active };
}

export function focus(tabs: Tabs, id: string): Tabs {
  return tabs.open.some((t) => t.id === id) ? { ...tabs, active: id } : tabs;
}

export function activeTab(tabs: Tabs): Tab | null {
  return tabs.open.find((t) => t.id === tabs.active) ?? null;
}
