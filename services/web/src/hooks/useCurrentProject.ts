import { useCallback, useEffect, useState } from 'react';

import type { Project } from '@/projects/types';

const REMEMBERED = 'oma.currentProject';

/**
 * The project being worked on.
 *
 * Remembered per device, like the layout will be (ADR-0011): which project you
 * had open is a property of this screen, not of the work.
 *
 * A remembered project that no longer exists is dropped rather than kept as a
 * dangling id, because everything downstream would then subscribe to a room
 * nothing publishes to and simply look broken.
 */
export function useCurrentProject(projects: Project[], loaded: boolean) {
  const [id, setId] = useState<string | null>(() => remembered());

  useEffect(() => {
    if (!loaded) return;
    if (id && !projects.some((p) => p.id === id)) setId(null);
  }, [id, loaded, projects]);

  const select = useCallback((next: string | null) => {
    setId(next);
    try {
      if (next) localStorage.setItem(REMEMBERED, next);
      else localStorage.removeItem(REMEMBERED);
    } catch {
      // A browser refusing storage is not a reason to stop working.
    }
  }, []);

  return { id, project: projects.find((p) => p.id === id) ?? null, select };
}

function remembered(): string | null {
  try {
    return localStorage.getItem(REMEMBERED);
  } catch {
    return null;
  }
}
