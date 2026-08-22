import { Button } from '@/components/ui/button';
import type { Project } from '@/projects/types';

type Props = {
  project: Project;
  busy: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

/**
 * Removal deletes the project's root directory, so this says which directory
 * before it happens. A confirmation that does not name what it is about is one
 * people learn to click through.
 */
export function ConfirmRemove({ project, busy, onConfirm, onCancel }: Props) {
  return (
    <div className="mb-4 rounded-app border border-danger/40 bg-danger/5 p-4">
      <p className="text-sm">
        Remove <span className="font-medium">{project.name}</span> and delete{' '}
        <span className="font-mono text-xs">{project.root}</span> with
        everything in it?
      </p>
      <div className="mt-3 flex gap-2">
        <Button size="sm" disabled={busy} onClick={onConfirm}>
          Delete it
        </Button>
        <Button size="sm" variant="outline" disabled={busy} onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
