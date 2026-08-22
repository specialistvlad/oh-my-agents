/** A project, as the api reports it. Mirrors services/api/internal/projects. */
export type Project = {
  id: string;
  name: string;
  root: string;
  created_at: string;
  updated_at: string;
};

/** The room project activity is published to. Shared scope, above any project. */
export const PROJECTS_ROOM = 'projects';

export const PROJECT_CREATED = 'project.created';
export const PROJECT_CHANGED = 'project.changed';
export const PROJECT_REMOVED = 'project.removed';
