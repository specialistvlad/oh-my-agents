import { useCallback, useState } from 'react';

import type { Project } from '@/projects/types';

/**
 * The page's own state: what is being typed, what is being renamed, and what
 * is one click away from being deleted.
 *
 * Removal deletes a directory (ADR-0010), so it is deliberately two steps.
 * Confirming names the project, because a confirmation that does not say what
 * it is about is one people learn to click through.
 */
export function useProjectsPage() {
  const [draft, setDraft] = useState('');
  const [itemDraft, setItemDraft] = useState('');
  const [editing, setEditing] = useState<Project | null>(null);
  const [editName, setEditName] = useState('');
  const [confirming, setConfirming] = useState<Project | null>(null);

  const startRename = useCallback((project: Project) => {
    setConfirming(null);
    setEditing(project);
    setEditName(project.name);
  }, []);

  const cancelRename = useCallback(() => setEditing(null), []);

  const startRemove = useCallback((project: Project) => {
    setEditing(null);
    setConfirming(project);
  }, []);

  const cancelRemove = useCallback(() => setConfirming(null), []);

  return {
    draft,
    setDraft,
    itemDraft,
    setItemDraft,
    editing,
    editName,
    setEditName,
    startRename,
    cancelRename,
    confirming,
    startRemove,
    cancelRemove,
  };
}
