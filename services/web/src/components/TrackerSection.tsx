import { CreateItem } from '@/components/CreateItem';
import { ItemList } from '@/components/ItemList';
import type { Project } from '@/projects/types';
import type { Item, Schema } from '@/tracker/types';

type Props = {
  project: Project | null;
  schema: Schema;
  items: Item[];
  loaded: boolean;
  busy: boolean;
  draft: string;
  onDraft: (value: string) => void;
  onCreate: (type: string) => void;
  onMove: (item: Item, status: string) => void;
  onRemove: (item: Item) => void;
};

/** The current project's items, or an invitation to pick one. */
export function TrackerSection({
  project,
  schema,
  items,
  loaded,
  busy,
  draft,
  onDraft,
  onCreate,
  onMove,
  onRemove,
}: Props) {
  return (
    <section>
      <h2 className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {project ? `Tracker — ${project.name}` : 'Tracker'}
      </h2>
      {project ? (
        <>
          <CreateItem
            value={draft}
            schema={schema}
            busy={busy}
            onChange={onDraft}
            onSubmit={onCreate}
          />
          <ItemList
            items={items}
            schema={schema}
            loaded={loaded}
            busy={busy}
            onMove={onMove}
            onRemove={onRemove}
          />
        </>
      ) : (
        <p className="py-3 text-sm text-muted-foreground">
          Select a project to see its items.
        </p>
      )}
    </section>
  );
}
