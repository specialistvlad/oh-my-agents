import type { RealtimeClient } from '@/realtime/client';

/**
 * The project lifecycle over the socket.
 *
 * Writes go the same way events come back, so a change made here reaches every
 * other open tab without either of them asking.
 */
export function createProject(
  client: RealtimeClient,
  name: string
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'project.create',
    id,
    name,
    idempotency,
  }));
}

export function renameProject(
  client: RealtimeClient,
  project: string,
  name: string
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'project.rename',
    id,
    project,
    name,
    idempotency,
  }));
}

/** Removes a project and deletes its root directory (ADR-0010). */
export function removeProject(
  client: RealtimeClient,
  project: string
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'project.remove',
    id,
    project,
    idempotency,
  }));
}
