import { Plus } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { Schema } from '@/tracker/types';

type Props = {
  value: string;
  schema: Schema;
  busy: boolean;
  onChange: (value: string) => void;
  onSubmit: (type: string) => void;
};

/**
 * Adds an item of the project's first type.
 *
 * One type is all a project starts with (a Task), so a picker would be a
 * control with one option. It arrives when a project has a second type.
 */
export function CreateItem({ value, schema, busy, onChange, onSubmit }: Props) {
  const type = schema.types[0];
  if (!type) return null;

  return (
    <form
      className="mb-4 flex gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (value.trim()) onSubmit(type.id);
      }}>
      <Input
        value={value}
        disabled={busy}
        placeholder={`New ${type.name.toLowerCase()}`}
        aria-label={`New ${type.name.toLowerCase()}`}
        onChange={(e) => onChange(e.target.value)}
      />
      <Button type="submit" disabled={busy || !value.trim()}>
        <Plus className="size-4" aria-hidden />
        Add
      </Button>
    </form>
  );
}
