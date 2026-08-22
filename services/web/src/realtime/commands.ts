import type { RealtimeClient } from './client';

/**
 * Settings writes over the socket.
 *
 * Separate from the client because the client is transport: it knows how to
 * send a command and match its reply, and nothing about what any command
 * means. These are the settings-shaped ones.
 */
export function setSetting(
  client: RealtimeClient,
  key: string,
  value: unknown
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'set',
    id,
    key,
    value,
    idempotency,
  }));
}

export function deleteSetting(
  client: RealtimeClient,
  key: string
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'delete',
    id,
    key,
    idempotency,
  }));
}
