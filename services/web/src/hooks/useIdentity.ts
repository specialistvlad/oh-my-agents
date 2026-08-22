import { useCallback, useEffect, useState } from 'react';

import { loadIdentity, saveIdentity } from '@/tracker/identity';
import type { Actor } from '@/tracker/types';

/**
 * Who this browser claims to be, and a way to change it.
 *
 * Read after mount rather than during render, so the value does not differ
 * between a server render and the first client one.
 */
export function useIdentity() {
  const [actor, setActor] = useState<Actor>(() => loadIdentity());
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');

  useEffect(() => setActor(loadIdentity()), []);

  return {
    actor,
    editing,
    draft,
    setDraft,
    start: useCallback(() => {
      setDraft(loadIdentity().id);
      setEditing(true);
    }, []),
    save: useCallback(() => {
      setActor(saveIdentity(draft));
      setEditing(false);
    }, [draft]),
    cancel: useCallback(() => setEditing(false), []),
  };
}
