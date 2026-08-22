import { Pencil, Plus, Trash2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/cn';
import type { Project } from '@/projects/types';
import { statusOf } from '@/tracker/types';
import type { Item, Schema } from '@/tracker/types';

type Props = {
  projects: Project[];
  currentProject: string | null;
  items: Item[];
  schema: Schema;
  selected: string | null;
  draft: string;
  busy: boolean;
  onDraft: (value: string) => void;
  onCreateProject: () => void;
  onRemoveProject: (project: Project) => void;
  onRenameProject: (project: Project) => void;
  onSelectProject: (project: Project) => void;
  onSelectItem: (item: Item) => void;
  onOpenItem: (item: Item) => void;
};

/**
 * The tree: projects, and the items inside the current one.
 *
 * A single click selects — which fills the inspector without opening
 * anything — and a double click opens a tab. That is the whole reason
 * selection and the active tab are separate (ADR-0011).
 */
export function ObjectsPanel({
  projects,
  currentProject,
  items,
  schema,
  selected,
  draft,
  busy,
  onDraft,
  onCreateProject,
  onRemoveProject,
  onRenameProject,
  onSelectProject,
  onSelectItem,
  onOpenItem,
}: Props) {
  return (
    <div className="p-2 text-sm">
      <form
        className="mb-2 flex gap-1"
        onSubmit={(e) => {
          e.preventDefault();
          if (draft.trim()) onCreateProject();
        }}>
        <Input
          value={draft}
          disabled={busy}
          placeholder="New project"
          aria-label="New project name"
          className="h-8"
          onChange={(e) => onDraft(e.target.value)}
        />
        <Button
          type="submit"
          size="sm"
          disabled={busy || !draft.trim()}
          aria-label="Create project">
          <Plus className="size-4" aria-hidden />
        </Button>
      </form>
      {projects.map((project) => (
        <div key={project.id}>
          <div className="flex items-center gap-1">
            <button
              type="button"
              className={cn(
                'min-w-0 flex-1 truncate rounded-app px-2 py-1 text-left font-medium',
                project.id === currentProject ? 'bg-muted' : 'hover:bg-muted/60'
              )}
              aria-pressed={project.id === currentProject}
              onClick={() => onSelectProject(project)}>
              {project.name}
            </button>
            <Button
              size="sm"
              variant="ghost"
              disabled={busy}
              aria-label={`Rename ${project.name}`}
              onClick={() => onRenameProject(project)}>
              <Pencil className="size-3.5" aria-hidden />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              disabled={busy}
              aria-label={`Remove ${project.name}`}
              onClick={() => onRemoveProject(project)}>
              <Trash2 className="size-3.5 text-danger" aria-hidden />
            </Button>
          </div>
          {project.id === currentProject ? (
            <ul className="mt-0.5 mb-2">
              {items.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    className={cn(
                      'flex w-full items-center gap-2 rounded-app py-1 pr-2 pl-5 text-left',
                      item.id === selected
                        ? 'bg-primary/10'
                        : 'hover:bg-muted/60'
                    )}
                    title="Click to inspect, double-click to open"
                    onClick={() => onSelectItem(item)}
                    onDoubleClick={() => onOpenItem(item)}>
                    <span className="size-1.5 shrink-0 rounded-full bg-muted-foreground" />
                    <span className="truncate">{item.title}</span>
                    <span className="ml-auto shrink-0 text-xs text-muted-foreground">
                      {statusOf(
                        schema.types.find((t) => t.id === item.type),
                        item.status
                      )?.name ?? ''}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ))}
    </div>
  );
}
