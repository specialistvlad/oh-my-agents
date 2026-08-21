import { backoffDelay } from './backoff';
import { decode, toEvent } from './frames';
import type { Inbound, Listener, Status } from './types';

/**
 * A realtime connection to the api.
 *
 * It reconnects on its own and re-joins the rooms it was in, so callers
 * declare what they want to watch once and never think about the socket
 * again. Nothing here polls: the only timer is the reconnect backoff.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private readonly rooms = new Set<string>();
  private readonly listeners = new Set<Listener>();
  private attempt = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = false;
  private nextId = 0;
  private status: Status = 'closed';
  private readonly pending = new Set<string>();

  constructor(private readonly url: string) {}

  /** Opens the connection, or does nothing if it is already open. */
  start(): void {
    this.stopped = false;
    if (!this.socket) this.open();
  }

  /** Closes the connection and stops reconnecting. */
  stop(): void {
    this.stopped = true;
    if (this.timer !== null) clearTimeout(this.timer);
    this.timer = null;
    this.socket?.close();
    this.socket = null;
    this.setStatus('closed');
  }

  /**
   * Subscribes to a room. Safe to call before the socket is open and safe to
   * call twice: the room is remembered and re-joined after any reconnect.
   */
  join(room: string): void {
    this.rooms.add(room);
    this.sendJoin(room);
  }

  leave(room: string): void {
    this.rooms.delete(room);
    this.send({ type: 'leave', id: this.id(), room });
  }

  /** Registers a listener and returns the function that removes it. */
  listen(listener: Listener): () => void {
    this.listeners.add(listener);
    listener.onStatus?.(this.status);
    return () => this.listeners.delete(listener);
  }

  private open(): void {
    this.setStatus(this.attempt === 0 ? 'connecting' : 'reconnecting');
    const socket = new WebSocket(this.url);
    this.socket = socket;

    socket.onopen = () => {
      this.attempt = 0;
      this.setStatus('open');
      // Re-join everything: the server knows nothing about a connection it
      // has not seen before, including one replacing a dropped socket.
      this.pending.clear();
      for (const room of this.rooms) {
        this.sendJoin(room);
      }
      if (this.rooms.size === 0) this.announceReady();
    };
    socket.onmessage = (message) => this.receive(message.data);
    socket.onclose = () => {
      if (this.socket === socket) this.socket = null;
      this.retry();
    };
    // An error is always followed by a close, which is where reconnecting is
    // handled; doing it here as well would open two sockets.
    socket.onerror = () => socket.close();
  }

  private retry(): void {
    if (this.stopped) return;
    this.attempt += 1;
    this.setStatus('reconnecting');
    this.timer = setTimeout(() => this.open(), backoffDelay(this.attempt));
  }

  private receive(raw: unknown): void {
    const frame = decode(raw);
    if (!frame) return;

    const event = toEvent(frame);
    if (event) {
      this.listeners.forEach((l) => l.onEvent?.(event));
      return;
    }
    if (frame.type === 'resync') {
      this.listeners.forEach((l) => l.onResync?.(frame.room ?? ''));
      return;
    }
    if (frame.type === 'ack' && frame.id && this.pending.delete(frame.id)) {
      if (this.pending.size === 0) this.announceReady();
    }
  }

  /** Sends a join and remembers it until the server acknowledges it. */
  private sendJoin(room: string): void {
    const id = this.id();
    this.pending.add(id);
    this.send({ type: 'join', id, room });
  }

  private announceReady(): void {
    this.listeners.forEach((l) => l.onReady?.());
  }

  private send(frame: Inbound): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(frame));
    }
    // Otherwise the room is already remembered and sent on the next open.
  }

  private setStatus(status: Status): void {
    if (status === this.status) return;
    this.status = status;
    this.listeners.forEach((l) => l.onStatus?.(status));
  }

  private id(): string {
    this.nextId += 1;
    return `c${this.nextId}`;
  }
}
