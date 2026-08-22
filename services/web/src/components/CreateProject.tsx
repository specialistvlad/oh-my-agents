import { Plus } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

type Props = {
  value: string;
  busy: boolean;
  onChange: (value: string) => void;
  onSubmit: () => void;
};

export function CreateProject({ value, busy, onChange, onSubmit }: Props) {
  return (
    <form
      className="mb-6 flex gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (value.trim()) onSubmit();
      }}>
      <Input
        value={value}
        disabled={busy}
        placeholder="New project name"
        aria-label="New project name"
        onChange={(e) => onChange(e.target.value)}
      />
      <Button type="submit" disabled={busy || !value.trim()}>
        <Plus className="size-4" aria-hidden />
        Create
      </Button>
    </form>
  );
}
