import { backoffDelay } from './backoff';
import { detach } from './detach';
import { decode, toEvent } from './frames';
import { Listeners } from './listeners';
import { Membership } from './membership';
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
  private readonly rooms = new Membership();
  private readonly listeners = new Listeners();
  private attempt = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = false;
  private status: Status = 'closed';

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

    // Detach before closing: the handlers belong to a socket nobody is
    // listening to any more, and leaving them attached is what lets a late
    // close event resurrect the connection.
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      detach(socket);
      socket.close();
    }
    this.rooms.reset();
    this.setStatus('closed');
  }

  /**
   * Subscribes to a room. Safe to call before the socket is open and safe to
   * call twice: the room is remembered and re-joined after any reconnect.
   */
  join(room: string): void {
    this.rooms.add(room);
    this.send({ type: 'join', id: this.rooms.joinId(), room });
  }

  leave(room: string): void {
    this.rooms.remove(room);
    this.send({ type: 'leave', id: this.rooms.commandId(), room });
  }

  /** Registers a listener and returns the function that removes it. */
  listen(listener: Listener): () => void {
    const remove = this.listeners.add(listener);
    listener.onStatus?.(this.status);
    return remove;
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
      this.rooms.reset();
      for (const room of this.rooms.rooms()) {
        this.send({ type: 'join', id: this.rooms.joinId(), room });
      }
      if (this.rooms.settled()) this.announceReady();
    };
    socket.onmessage = (message) => this.receive(message.data);
    socket.onclose = () => {
      // Only the socket we are actually using may trigger a reconnect. A
      // close event arrives asynchronously, so one from a socket already
      // replaced or stopped can land after the client has moved on — and
      // reconnecting from that leaves two live sockets delivering every
      // event twice. A remount does exactly this.
      if (this.socket !== socket) return;
      this.socket = null;
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
      this.listeners.notify((l) => l.onEvent?.(event));
      return;
    }
    if (frame.type === 'resync') {
      this.listeners.notify((l) => l.onResync?.(frame.room ?? ''));
      return;
    }
    if (frame.type === 'ack' && frame.id && this.rooms.acknowledge(frame.id)) {
      this.announceReady();
    }
  }

  private announceReady(): void {
    this.listeners.notify((l) => l.onReady?.());
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
    this.listeners.notify((l) => l.onStatus?.(status));
  }
}
