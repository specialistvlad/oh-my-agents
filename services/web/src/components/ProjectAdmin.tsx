import { ConfirmRemove } from '@/components/ConfirmRemove';
import { RenameProject } from '@/components/RenameProject';
import type { Project } from '@/projects/types';

export type ProjectAdminProps = {
  /** Being renamed, if anything is. */
  editing: Project | null;
  editName: string;
  /** One click from deletion, which takes its directory with it. */
  confirming: Project | null;
  busy: boolean;
  onEditName: (value: string) => void;
  onRename: () => void;
  onCancelRename: () => void;
  onRemove: () => void;
  onCancelRemove: () => void;
};

/**
 * Whichever project dialog is open, or nothing.
 *
 * The two are one concern — editing a project that already exists — so they
 * travel together rather than as eight props threaded separately through the
 * shell.
 */
export function ProjectAdmin(p: ProjectAdminProps) {
  if (p.editing) {
    return (
      <RenameProject
        project={p.editing}
        value={p.editName}
        busy={p.busy}
        onChange={p.onEditName}
        onSubmit={p.onRename}
        onCancel={p.onCancelRename}
      />
    );
  }
  if (p.confirming) {
    return (
      <ConfirmRemove
        project={p.confirming}
        busy={p.busy}
        onConfirm={p.onRemove}
        onCancel={p.onCancelRemove}
      />
    );
  }
  return null;
}
