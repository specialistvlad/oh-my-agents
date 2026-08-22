import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { Actor } from '@/tracker/types';

type Props = {
  actor: Actor;
  editing: boolean;
  draft: string;
  onDraft: (value: string) => void;
  onStart: () => void;
  onSave: () => void;
  onCancel: () => void;
};

/**
 * The name this browser puts on everything it writes.
 *
 * Deliberately plain, and deliberately editable in one click. There is no
 * authentication (ADR-0012), so this is a claim rather than an identity, and
 * dressing it up as an account would say something untrue.
 */
export function IdentityBadge({
  actor,
  editing,
  draft,
  onDraft,
  onStart,
  onSave,
  onCancel,
}: Props) {
  if (!editing) {
    return (
      <Button
        size="sm"
        variant="ghost"
        title="The name your changes are recorded under. Not verified."
        onClick={onStart}>
        {actor.id}
      </Button>
    );
  }
  return (
    <form
      className="flex items-center gap-1"
      onSubmit={(e) => {
        e.preventDefault();
        onSave();
      }}>
      <Input
        autoFocus
        value={draft}
        aria-label="Your name"
        className="h-7 w-32"
        onChange={(e) => onDraft(e.target.value)}
        onBlur={onSave}
        onKeyDown={(e) => {
          if (e.key === 'Escape') onCancel();
        }}
      />
    </form>
  );
}
