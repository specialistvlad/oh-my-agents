import { Pencil, Trash2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import type { Project } from '@/projects/types';

type Props = {
  project: Project;
  busy: boolean;
  onRename: (project: Project) => void;
  onRemove: (project: Project) => void;
};

/** One project: what it is called, where it lives, and what can be done to it. */
export function ProjectRow({ project, busy, onRename, onRemove }: Props) {
  return (
    <li className="flex items-center gap-3 border-b border-border py-3 last:border-0">
      <div className="min-w-0 flex-1">
        <div className="truncate font-medium">{project.name}</div>
        <div className="truncate font-mono text-xs text-muted-foreground">
          {project.id} · {project.root}
        </div>
      </div>
      <Button
        size="sm"
        variant="ghost"
        disabled={busy}
        aria-label={`Rename ${project.name}`}
        onClick={() => onRename(project)}>
        <Pencil className="size-4" aria-hidden />
      </Button>
      <Button
        size="sm"
        variant="ghost"
        disabled={busy}
        aria-label={`Remove ${project.name}`}
        onClick={() => onRemove(project)}>
        <Trash2 className="size-4 text-danger" aria-hidden />
      </Button>
    </li>
  );
}
