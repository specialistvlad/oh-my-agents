// The wire protocol, mirroring services/api/internal/realtimews/protocol.go.
// Kept as plain types rather than generated, because it is small and a
// generator would be more machinery than the thing it generates.

export type Inbound =
  | { type: 'join'; id: string; room: string }
  | { type: 'leave'; id: string; room: string }
  | { type: 'ping'; id: string };

export type Outbound = {
  type: 'welcome' | 'ack' | 'error' | 'event' | 'resync' | 'pong';
  id?: string;
  room?: string;
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
  /** The connection missed messages; re-read from the server. */
  onResync?: (room: string) => void;
};
