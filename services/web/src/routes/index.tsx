import { createFileRoute } from '@tanstack/react-router';

import { ConfirmRemove } from '@/components/ConfirmRemove';
import { ConnectionBadge } from '@/components/ConnectionBadge';
import { CreateProject } from '@/components/CreateProject';
import { ProjectList } from '@/components/ProjectList';
import { RenameProject } from '@/components/RenameProject';
import { TrackerSection } from '@/components/TrackerSection';
import { Separator } from '@/components/ui/separator';
import { configuration } from '@/core/configuration';
import { useCurrentProject } from '@/hooks/useCurrentProject';
import { useProjectWriter } from '@/hooks/useProjectWriter';
import { useProjects } from '@/hooks/useProjects';
import { useProjectsPage } from '@/hooks/useProjectsPage';
import { useRealtime } from '@/hooks/useRealtime';
import { useTracker } from '@/hooks/useTracker';
import { useTrackerWriter } from '@/hooks/useTrackerWriter';
import { PROJECTS_ROOM, projectRoom } from '@/projects/types';

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
  const {
    projects,
    error: listError,
    loaded,
  } = useProjects(configuration.apiUrl, generation, events);
  const current = useCurrentProject(projects, loaded);
  const projectWriter = useProjectWriter(client);
  const tracker = useTracker(
    configuration.apiUrl,
    current.id,
    generation,
    events
  );
  const items = useTrackerWriter(client, current.id);
  const page = useProjectsPage();

  const error =
    projectWriter.error ?? items.error ?? tracker.error ?? listError;
  const busy = projectWriter.busy || items.busy;

  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <div className="mb-1 flex items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">oh-my-agents</h1>
        <ConnectionBadge status={status} />
      </div>
      <p className="mb-6 text-sm text-muted-foreground">
        One socket carries writes out and events back. State is fetched when the
        rooms are joined; after that nothing here polls.
      </p>

      <CreateProject
        value={page.draft}
        busy={busy}
        onChange={page.setDraft}
        onSubmit={() => {
          void projectWriter.create(page.draft).then(() => page.setDraft(''));
        }}
      />

      {page.editing ? (
        <RenameProject
          project={page.editing}
          value={page.editName}
          busy={busy}
          onChange={page.setEditName}
          onSubmit={() => {
            const target = page.editing;
            if (target) {
              void projectWriter
                .rename(target.id, page.editName)
                .then(page.cancelRename);
            }
          }}
          onCancel={page.cancelRename}
        />
      ) : null}

      {page.confirming ? (
        <ConfirmRemove
          project={page.confirming}
          busy={busy}
          onConfirm={() => {
            const target = page.confirming;
            if (target)
              void projectWriter.remove(target.id).then(page.cancelRemove);
          }}
          onCancel={page.cancelRemove}
        />
      ) : null}

      {error ? <p className="mb-4 text-sm text-danger">{error}</p> : null}

      <ProjectList
        projects={projects}
        selectedID={current.id}
        loaded={loaded}
        busy={busy}
        onSelect={(p) => current.select(current.id === p.id ? null : p.id)}
        onRename={page.startRename}
        onRemove={page.startRemove}
      />

      <Separator className="my-6" />

      <TrackerSection
        project={current.project}
        schema={tracker.schema}
        items={tracker.items}
        loaded={tracker.loaded}
        busy={busy}
        draft={page.itemDraft}
        onDraft={page.setItemDraft}
        onCreate={(type) => {
          void items
            .create(type, page.itemDraft)
            .then(() => page.setItemDraft(''));
        }}
        onMove={(item, status) =>
          void items.move(item.id, item.version, status)
        }
        onRemove={(item) => void items.remove(item.id, item.version)}
      />
    </main>
  );
}
