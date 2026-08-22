import { Check, Pencil, Trash2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/cn';
import type { Project } from '@/projects/types';

type Props = {
  project: Project;
  selected: boolean;
  busy: boolean;
  onSelect: (project: Project) => void;
  onRename: (project: Project) => void;
  onRemove: (project: Project) => void;
};

/** One project: what it is called, where it lives, and what can be done to it. */
export function ProjectRow({
  project,
  selected,
  busy,
  onSelect,
  onRename,
  onRemove,
}: Props) {
  return (
    <li
      className={cn(
        'flex items-center gap-3 border-b border-border py-3 last:border-0',
        selected && 'bg-muted/50'
      )}>
      <button
        type="button"
        className="min-w-0 flex-1 text-left"
        aria-pressed={selected}
        onClick={() => onSelect(project)}>
        <span className="flex items-center gap-2 truncate font-medium">
          {selected ? (
            <Check className="size-4 text-primary" aria-hidden />
          ) : null}
          {project.name}
        </span>
        <span className="block truncate font-mono text-xs text-muted-foreground">
          {project.id} · {project.root}
        </span>
      </button>
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
