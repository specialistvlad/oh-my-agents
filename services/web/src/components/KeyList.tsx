import { Badge } from '@/components/ui/badge';

/**
 * The settings that exist, as of the last fetch.
 *
 * A failed fetch is shown as a failure. Rendering "none stored" when the
 * request never succeeded says something false about the server, and that is
 * exactly the state a broken API is in.
 */
export function KeyList({
  keys,
  error,
}: {
  keys: string[];
  error: string | null;
}) {
  if (error) {
    return (
      <p className="py-2 text-sm text-danger">
        Could not load settings: {error}
      </p>
    );
  }
  if (keys.length === 0) {
    return (
      <p className="py-2 text-sm text-muted-foreground">No settings stored.</p>
    );
  }
  return (
    <div className="flex flex-wrap gap-2 py-2">
      {keys.map((key) => (
        <Badge key={key} className="font-mono">
          {key}
        </Badge>
      ))}
    </div>
  );
}
