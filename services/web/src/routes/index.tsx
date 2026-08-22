import { createFileRoute } from '@tanstack/react-router';

import { Centre } from '@/components/workspace/Centre';
import { Inspector } from '@/components/workspace/Inspector';
import { ObjectsPanel } from '@/components/workspace/ObjectsPanel';
import { Workspace } from '@/components/workspace/Workspace';
import { configuration } from '@/core/configuration';
import { useCurrentProject } from '@/hooks/useCurrentProject';
import { useIdentity } from '@/hooks/useIdentity';
import { useProjectAdmin } from '@/hooks/useProjectAdmin';
import { useProjects } from '@/hooks/useProjects';
import { useRealtime } from '@/hooks/useRealtime';
import { useTombstones } from '@/hooks/useTombstones';
import { useTracker } from '@/hooks/useTracker';
import { useTrackerWriter } from '@/hooks/useTrackerWriter';
import { useWorkspace } from '@/hooks/useWorkspace';
import { PROJECTS_ROOM, projectRoom } from '@/projects/types';
import { activeTab } from '@/workspace/tabs';

export const Route = createFileRoute('/')({ component: Index });

function Index() {
  // Which project is current is remembered per device; reading it before the
  // list arrives is what lets the right room be joined on the first connect
  // rather than after a round trip.
  const remembered = useCurrentProject([], false);
  const rooms = remembered.id
    ? [PROJECTS_ROOM, projectRoom(remembered.id)]
    : [PROJECTS_ROOM];

  const { client, status, events, generation } = useRealtime(
    configuration.apiUrl,
    rooms
  );
  const { projects, loaded } = useProjects(
    configuration.apiUrl,
    generation,
    events
  );
  const current = useCurrentProject(projects, loaded);
  const tracker = useTracker(
    configuration.apiUrl,
    current.id,
    generation,
    events
  );
  const writer = useTrackerWriter(client, current.id);
  const admin = useProjectAdmin(client);
  const shell = useWorkspace();
  const me = useIdentity();

  useTombstones(
    shell,
    shell.tabs.open.map((t) => t.id),
    tracker.items,
    tracker.loaded
  );

  const open = activeTab(shell.tabs);
  const openItem = tracker.items.find((i) => i.id === open?.id) ?? null;
  const selectedItem =
    tracker.items.find((i) => i.id === shell.selected) ?? null;

  return (
    <Workspace
      layout={shell.layout}
      onPanel={shell.selectPanel}
      onResizeLeft={shell.resizeLeft}
      onResizeRight={shell.resizeRight}
      left={
        <ObjectsPanel
          projects={projects}
          currentProject={current.id}
          items={tracker.items}
          schema={tracker.schema}
          selected={shell.selected}
          draft={admin.draft}
          busy={admin.busy}
          onDraft={admin.setDraft}
          onCreateProject={admin.create}
          onRemoveProject={admin.startRemove}
          onRenameProject={admin.startRename}
          onSelectProject={(p) =>
            current.select(current.id === p.id ? null : p.id)
          }
          onSelectItem={(item) => shell.select(item.id)}
          onOpenItem={(item) =>
            shell.openTab({ id: item.id, title: item.title })
          }
        />
      }
      centre={
        <Centre
          status={status}
          tabs={shell.tabs.open}
          active={shell.tabs.active}
          openItem={openItem}
          openIsGone={Boolean(open?.gone)}
          items={tracker.items}
          schema={tracker.schema}
          loaded={tracker.loaded}
          busy={writer.busy}
          admin={admin.dialog}
          draft={shell.draft}
          hasProject={Boolean(current.project)}
          onDraft={shell.setDraft}
          onCreate={(type) => {
            void writer
              .create(type, shell.draft)
              .then(() => shell.setDraft(''));
          }}
          onMove={(item, next) => void writer.move(item.id, item.version, next)}
          onRemove={(item) => void writer.remove(item.id, item.version)}
          onFocusTab={shell.focusTab}
          onCloseTab={shell.closeTab}
          onToggleInspector={shell.toggleInspector}
          identity={{
            actor: me.actor,
            editing: me.editing,
            draft: me.draft,
            onDraft: me.setDraft,
            onStart: me.start,
            onSave: me.save,
            onCancel: me.cancel,
          }}
        />
      }
      right={
        <Inspector
          item={selectedItem}
          missing={Boolean(shell.selected) && tracker.loaded && !selectedItem}
          schema={tracker.schema}
          busy={writer.busy}
          onMove={(item, next) => void writer.move(item.id, item.version, next)}
        />
      }
    />
  );
}
