import { StatusChip } from '@/components/StatusChip';
import { Button } from '@/components/ui/button';
import { formatValue } from '@/tracker/format';
import { fieldsOf, movesFrom, statusOf } from '@/tracker/types';
import type { Item, Schema } from '@/tracker/types';

type Props = {
  item: Item | null;
  /** Selected, but the object is gone. */
  missing: boolean;
  schema: Schema;
  busy: boolean;
  onMove: (item: Item, status: string) => void;
};

/**
 * What is selected, which is not necessarily what is open (ADR-0011).
 *
 * A selection whose object has been deleted reports that rather than emptying,
 * for the same reason a tab does: silently showing nothing looks like a bug
 * and leaves the person wondering what they clicked.
 */
export function Inspector({ item, missing, schema, busy, onMove }: Props) {
  if (missing) {
    return (
      <Panel>
        <p className="text-sm text-danger">
          This item has been deleted. Select something else.
        </p>
      </Panel>
    );
  }
  if (!item) {
    return (
      <Panel>
        <p className="text-sm text-muted-foreground">
          Select something to inspect it. Selecting does not open it.
        </p>
      </Panel>
    );
  }
  const type = schema.types.find((t) => t.id === item.type);
  return (
    <Panel>
      <h2 className="mb-3 text-sm font-medium">{item.title}</h2>
      <Row label="Status">
        <StatusChip status={statusOf(type, item.status)} />
      </Row>
      <Row label="Type">{type?.name ?? item.type}</Row>
      <Row label="Version">{item.version}</Row>
      <Row label="ID">
        <span className="font-mono text-xs break-all">{item.id}</span>
      </Row>
      {/*
        The fields this item's type declares, whether or not the item holds a
        value for one. Showing the empty ones is what makes a configured field
        discoverable — a field nobody has filled in is invisible otherwise.
      */}
      {fieldsOf(type, item).map(({ def, value }) => (
        <Row key={def.id} label={def.name}>
          {formatValue(value)}
        </Row>
      ))}
      {type ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {movesFrom(type, item.status).map((status) => (
            <Button
              key={status.id}
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => onMove(item, status.id)}>
              {status.name}
            </Button>
          ))}
        </div>
      ) : null}
    </Panel>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return <div className="p-4">{children}</div>;
}

function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex gap-3 border-b border-border py-1.5 text-sm last:border-0">
      <span className="w-16 shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0">{children}</span>
    </div>
  );
}
