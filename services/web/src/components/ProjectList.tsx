import { ProjectRow } from '@/components/ProjectRow';
import type { Project } from '@/projects/types';

type Props = {
  projects: Project[];
  loaded: boolean;
  busy: boolean;
  onRename: (project: Project) => void;
  onRemove: (project: Project) => void;
};

export function ProjectList({
  projects,
  loaded,
  busy,
  onRename,
  onRemove,
}: Props) {
  if (!loaded) {
    return <p className="py-3 text-sm text-muted-foreground">Loading…</p>;
  }
  if (projects.length === 0) {
    return (
      <p className="py-3 text-sm text-muted-foreground">
        No projects yet. Create one — then open this page in a second tab and
        watch both stay in step.
      </p>
    );
  }
  return (
    <ul>
      {projects.map((project) => (
        <ProjectRow
          key={project.id}
          project={project}
          busy={busy}
          onRename={onRename}
          onRemove={onRemove}
        />
      ))}
    </ul>
  );
}
