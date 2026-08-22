import { CreateItem } from '@/components/CreateItem';
import { ItemList } from '@/components/ItemList';
import { Board } from '@/components/board/Board';
import type { CentreProps } from '@/components/workspace/Centre';

/**
 * What the centre shows: the open item, the board, or the list.
 *
 * Split from the chrome around it — the header, the tabs — because those stay
 * put while this changes with every view and every tab.
 */
export function CentreBody(props: CentreProps) {
  if (props.openIsGone) {
    return (
      <p className="text-sm text-danger">
        This item has been deleted. Close the tab when you are done with it.
      </p>
    );
  }
  if (props.openItem) {
    return (
      <article>
        <h1 className="mb-2 text-xl font-semibold">{props.openItem.title}</h1>
        <p className="text-sm whitespace-pre-wrap text-muted-foreground">
          {props.openItem.body || 'No description.'}
        </p>
      </article>
    );
  }
  if (!props.hasProject) {
    return (
      <p className="text-sm text-muted-foreground">
        Select a project in the objects panel.
      </p>
    );
  }
  if (props.view === 'board') {
    return (
      <div className="flex h-full flex-col gap-3">
        <CreateItem
          value={props.draft}
          schema={props.schema}
          busy={props.busy}
          onChange={props.onDraft}
          onSubmit={props.onCreate}
        />
        <div className="min-h-0 flex-1">
          <Board
            schema={props.schema}
            items={props.items}
            loaded={props.loaded}
            selected={props.selected}
            drag={props.drag}
            onSelect={props.onSelect}
            onOpen={props.onOpen}
            onDrop={props.onDropCard}
          />
        </div>
      </div>
    );
  }
  return (
    <>
      <CreateItem
        value={props.draft}
        schema={props.schema}
        busy={props.busy}
        onChange={props.onDraft}
        onSubmit={props.onCreate}
      />
      <ItemList
        items={props.items}
        schema={props.schema}
        loaded={props.loaded}
        busy={props.busy}
        onMove={props.onMove}
        onRemove={props.onRemove}
      />
    </>
  );
}
