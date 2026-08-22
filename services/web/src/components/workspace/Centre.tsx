import { PanelRight } from 'lucide-react';

import { ConnectionBadge } from '@/components/ConnectionBadge';
import { CreateItem } from '@/components/CreateItem';
import { ItemList } from '@/components/ItemList';
import { ProjectAdmin } from '@/components/ProjectAdmin';
import type { ProjectAdminProps } from '@/components/ProjectAdmin';
import { Button } from '@/components/ui/button';
import { TabStrip } from '@/components/workspace/TabStrip';
import type { Status } from '@/realtime/types';
import type { Item, Schema } from '@/tracker/types';
import type { Tab } from '@/workspace/tabs';

type Props = {
  status: Status;
  tabs: Tab[];
  active: string | null;
  openItem: Item | null;
  openIsGone: boolean;
  items: Item[];
  schema: Schema;
  loaded: boolean;
  busy: boolean;
  draft: string;
  hasProject: boolean;
  /** Whichever project dialog is open, if any. */
  admin: ProjectAdminProps;
  onDraft: (value: string) => void;
  onCreate: (type: string) => void;
  onMove: (item: Item, status: string) => void;
  onRemove: (item: Item) => void;
  onFocusTab: (id: string) => void;
  onCloseTab: (id: string) => void;
  onToggleInspector: () => void;
};

/** The middle column: the tab strip, and whatever the active tab holds. */
export function Centre(props: Props) {
  return (
    <>
      <header className="flex items-center gap-3 border-b border-border px-4 py-2">
        <span className="text-sm font-semibold">oh-my-agents</span>
        <ConnectionBadge status={props.status} />
        <Button
          size="sm"
          variant="ghost"
          className="ml-auto"
          aria-label="Toggle the inspector"
          onClick={props.onToggleInspector}>
          <PanelRight className="size-4" aria-hidden />
        </Button>
      </header>
      <TabStrip
        tabs={props.tabs}
        active={props.active}
        onFocus={props.onFocusTab}
        onClose={props.onCloseTab}
      />
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <ProjectAdmin {...props.admin} />
        <Body {...props} />
      </div>
    </>
  );
}

function Body(props: Props) {
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
