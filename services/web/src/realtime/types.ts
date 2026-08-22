// The wire protocol, mirroring services/api/internal/realtimews/protocol.go.
// Kept as plain types rather than generated, because it is small and a
// generator would be more machinery than the thing it generates.

export type Inbound =
  | { type: 'join'; id: string; room: string }
  | { type: 'leave'; id: string; room: string }
  | { type: 'ping'; id: string }
  | {
      type: 'set';
      id: string;
      key: string;
      value: unknown;
      idempotency: string;
    }
  | { type: 'delete'; id: string; key: string; idempotency: string }
  | { type: 'project.create'; id: string; name: string; idempotency: string }
  | {
      type: 'project.rename';
      id: string;
      project: string;
      name: string;
      idempotency: string;
    }
  | {
      type: 'project.repoint';
      id: string;
      project: string;
      root: string;
      idempotency: string;
    }
  | {
      type: 'project.remove';
      id: string;
      project: string;
      idempotency: string;
    }
  | {
      type: 'item.create';
      id: string;
      project: string;
      idempotency: string;
      body: unknown;
    }
  | {
      type: 'item.update';
      id: string;
      project: string;
      item: string;
      version: number;
      idempotency: string;
      body: unknown;
    }
  | {
      type: 'item.delete';
      id: string;
      project: string;
      item: string;
      version: number;
      idempotency: string;
    };

export type Outbound = {
  type: 'welcome' | 'ack' | 'error' | 'event' | 'resync' | 'pong';
  id?: string;
  room?: string;
  key?: string;
  status?: number;
  seq?: number;
  kind?: string;
  data?: unknown;
  error?: string;
};

/** What happened, delivered without anyone asking. */
export type RealtimeEvent = {
  room: string;
  seq: number;
  kind: string;
  data: unknown;
};

/**
 * Connection state, as a client actually experiences it.
 *
 * `reconnecting` is separate from `connecting` on purpose: the first
 * connection failing and a live connection dropping look identical to the
 * code and completely different to a person watching the screen.
 */
export type Status = 'connecting' | 'open' | 'reconnecting' | 'closed';

export type Listener = {
  onEvent?: (event: RealtimeEvent) => void;
  onStatus?: (status: Status) => void;
  /**
   * Every requested room is now joined, so anything that happens from here
   * will be delivered. This is the signal to fetch current state: fetching
   * before it would leave a window where a change is missed by both the
   * fetch and the socket.
   */
  onReady?: () => void;
  /**
   * The connection missed messages. Fetch the current state of whatever is
   * on screen again; nothing is replayed.
   */
  onResync?: (room: string) => void;
};
