/** A project, as the api reports it. Mirrors services/api/internal/projects. */
export type Project = {
  id: string;
  name: string;
  root: string;
  created_at: string;
  updated_at: string;
};

/**
 * The room the project list is published to. Shared scope, above any project,
 * because a client watching the list is not yet watching a project.
 */
export const PROJECTS_ROOM = 'projects';

/**
 * The room everything inside one project is published to (ADR-0009).
 *
 * Spelled here rather than at each call site: a room name is a contract with
 * the server, and two spellings of the same room is a bug nobody sees until a
 * client is silently deaf.
 */
export function projectRoom(id: string): string {
  return `project:${id}`;
}

export const PROJECT_CREATED = 'project.created';
export const PROJECT_CHANGED = 'project.changed';
export const PROJECT_REMOVED = 'project.removed';
