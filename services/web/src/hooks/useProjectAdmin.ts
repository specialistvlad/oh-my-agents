import { useCallback, useState } from 'react';

import type { ProjectAdminProps } from '@/components/ProjectAdmin';
import { useProjectWriter } from '@/hooks/useProjectWriter';
import type { Project } from '@/projects/types';
import type { RealtimeClient } from '@/realtime/client';

/**
 * Creating, renaming and removing projects.
 *
 * The writes and the dialogs they belong to are one concern, so they live
 * together: keeping them apart meant threading a dozen props through the
 * shell to reunite them at the far end.
 *
 * Removal is deliberately two steps, because it deletes the project's root
 * directory (ADR-0010).
 */
export function useProjectAdmin(client: RealtimeClient) {
  const writer = useProjectWriter(client);
  const [draft, setDraft] = useState('');
  const [editing, setEditing] = useState<Project | null>(null);
  const [editName, setEditName] = useState('');
  const [confirming, setConfirming] = useState<Project | null>(null);

  const startRename = useCallback((project: Project) => {
    setConfirming(null);
    setEditing(project);
    setEditName(project.name);
  }, []);

  const startRemove = useCallback((project: Project) => {
    setEditing(null);
    setConfirming(project);
  }, []);

  const cancelRename = useCallback(() => setEditing(null), []);
  const cancelRemove = useCallback(() => setConfirming(null), []);

  const dialog: ProjectAdminProps = {
    editing,
    editName,
    confirming,
    busy: writer.busy,
    onEditName: setEditName,
    onRename: () => {
      if (editing) void writer.rename(editing.id, editName).then(cancelRename);
    },
    onCancelRename: cancelRename,
    onRemove: () => {
      if (confirming) void writer.remove(confirming.id).then(cancelRemove);
    },
    onCancelRemove: cancelRemove,
  };

  return {
    busy: writer.busy,
    error: writer.error,
    draft,
    setDraft,
    create: useCallback(() => {
      void writer.create(draft).then(() => setDraft(''));
    }, [writer, draft]),
    startRename,
    startRemove,
    dialog,
  };
}
