import { Columns3, List, PanelRight } from 'lucide-react';
import type { ComponentProps } from 'react';

import { ConnectionBadge } from '@/components/ConnectionBadge';
import { IdentityBadge } from '@/components/IdentityBadge';
import { ProjectAdmin } from '@/components/ProjectAdmin';
import type { ProjectAdminProps } from '@/components/ProjectAdmin';
import { Button } from '@/components/ui/button';
import { CentreBody } from '@/components/workspace/CentreBody';
import { TabStrip } from '@/components/workspace/TabStrip';
import type { Drag } from '@/hooks/useBoardDrag';
import type { Status } from '@/realtime/types';
import type { Drop } from '@/tracker/board';
import type { Item, Schema } from '@/tracker/types';
import type { View } from '@/workspace/layout';
import type { Tab } from '@/workspace/tabs';

export type CentreProps = {
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
  identity: ComponentProps<typeof IdentityBadge>;
  view: View;
  onView: (view: View) => void;
  /** Board drag state, shared between the card that starts it and the column
   * that ends it. */
  drag: Drag;
  selected: string | null;
  onSelect: (item: Item) => void;
  onOpen: (item: Item) => void;
  onDropCard: (item: Item, drop: Drop) => void;
};

/** The middle column: the tab strip, and whatever the active tab holds. */
export function Centre(props: CentreProps) {
  return (
    <>
      <header className="flex items-center gap-3 border-b border-border px-4 py-2">
        <span className="text-sm font-semibold">oh-my-agents</span>
        <ConnectionBadge status={props.status} />
        <div className="ml-auto flex items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            aria-label="Board view"
            aria-pressed={props.view === 'board'}
            onClick={() => props.onView('board')}>
            <Columns3 className="size-4" aria-hidden />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            aria-label="List view"
            aria-pressed={props.view === 'list'}
            onClick={() => props.onView('list')}>
            <List className="size-4" aria-hidden />
          </Button>
          <IdentityBadge {...props.identity} />
          <Button
            size="sm"
            variant="ghost"
            aria-label="Toggle the inspector"
            onClick={props.onToggleInspector}>
            <PanelRight className="size-4" aria-hidden />
          </Button>
        </div>
      </header>
      <TabStrip
        tabs={props.tabs}
        active={props.active}
        onFocus={props.onFocusTab}
        onClose={props.onCloseTab}
      />
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <ProjectAdmin {...props.admin} />
        <CentreBody {...props} />
      </div>
    </>
  );
}
