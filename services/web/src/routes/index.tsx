import { createFileRoute } from '@tanstack/react-router';

import { ConfirmRemove } from '@/components/ConfirmRemove';
import { ConnectionBadge } from '@/components/ConnectionBadge';
import { CreateProject } from '@/components/CreateProject';
import { EventList } from '@/components/EventList';
import { ProjectList } from '@/components/ProjectList';
import { RenameProject } from '@/components/RenameProject';
import { Separator } from '@/components/ui/separator';
import { configuration } from '@/core/configuration';
import { useCurrentProject } from '@/hooks/useCurrentProject';
import { useProjectWriter } from '@/hooks/useProjectWriter';
import { useProjects } from '@/hooks/useProjects';
import { useProjectsPage } from '@/hooks/useProjectsPage';
import { useRealtime } from '@/hooks/useRealtime';
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
  const {
    create,
    rename,
    remove,
    busy,
    error: writeError,
  } = useProjectWriter(client);
  const page = useProjectsPage();

  const error = writeError ?? listError;

  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <div className="mb-1 flex items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">Projects</h1>
        <ConnectionBadge status={status} />
      </div>
      <p className="mb-6 text-sm text-muted-foreground">
        Changes go out over one socket and come back to every open tab. The list
        is fetched once on connect; after that nothing here polls.
      </p>

      <CreateProject
        value={page.draft}
        busy={busy}
        onChange={page.setDraft}
        onSubmit={() => {
          void create(page.draft).then(() => page.setDraft(''));
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
            if (target)
              void rename(target.id, page.editName).then(page.cancelRename);
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
            if (target) void remove(target.id).then(page.cancelRemove);
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

      <h2 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
        Live events
      </h2>
      <EventList events={events} />
    </main>
  );
}
