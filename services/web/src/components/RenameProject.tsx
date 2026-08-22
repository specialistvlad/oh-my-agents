import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { Project } from '@/projects/types';

type Props = {
  project: Project;
  value: string;
  busy: boolean;
  onChange: (value: string) => void;
  onSubmit: () => void;
  onCancel: () => void;
};

export function RenameProject({
  project,
  value,
  busy,
  onChange,
  onSubmit,
  onCancel,
}: Props) {
  return (
    <form
      className="mb-4 rounded-app border border-border bg-muted/40 p-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (value.trim()) onSubmit();
      }}>
      <p className="mb-2 text-sm text-muted-foreground">
        Renaming <span className="font-mono text-xs">{project.id}</span>. The id
        never changes, so nothing that points at this project breaks.
      </p>
      <div className="flex gap-2">
        <Input
          value={value}
          disabled={busy}
          aria-label="New name"
          onChange={(e) => onChange(e.target.value)}
        />
        <Button type="submit" size="sm" disabled={busy || !value.trim()}>
          Save
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={busy}
          onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
