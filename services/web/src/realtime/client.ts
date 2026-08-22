import { decode, toEvent } from './frames';
import { Listeners } from './listeners';
import { Membership } from './membership';
import { Pending } from './pending';
import { Socket } from './socket';
import type { Inbound, Listener, Status } from './types';

/**
 * A realtime connection to the api.
 *
 * It reconnects on its own and re-joins the rooms it was in, so callers
 * declare what they want to watch once and never think about the socket
 * again. Nothing here polls: the only timer is the reconnect backoff.
 */
export class RealtimeClient {
  private readonly socket: Socket;
  private readonly rooms = new Membership();
  private readonly listeners = new Listeners();
  private readonly awaiting = new Pending();
  private readonly stamp = `${Date.now()}`;
  private status: Status = 'closed';

  constructor(url: string) {
    this.socket = new Socket(url, {
      onOpen: () => this.rejoin(),
      onFrame: (raw) => this.receive(raw),
      onDrop: () => this.awaiting.abandon(),
      onStatus: (status) => {
        this.status = status;
        this.listeners.notify((l) => l.onStatus?.(status));
      },
    });
  }

  start(): void {
    this.socket.start();
  }

  stop(): void {
    this.rooms.reset();
    this.socket.stop();
  }

  /**
   * Subscribes to a room. Safe before the socket is open and safe twice: the
   * room is remembered and re-joined after any reconnect.
   */
  join(room: string): void {
    this.rooms.add(room);
    this.socket.send({ type: 'join', id: this.rooms.joinId(), room });
  }

  leave(room: string): void {
    this.rooms.remove(room);
    this.socket.send({ type: 'leave', id: this.rooms.commandId(), room });
  }

  /** Registers a listener and returns the function that removes it. */
  listen(listener: Listener): () => void {
    const remove = this.listeners.add(listener);
    listener.onStatus?.(this.status);
    return remove;
  }

  /**
   * Sends a command and resolves when the reply carrying its id arrives.
   *
   * The builder is handed an id and an idempotency key: the id pairs the
   * reply to the call, and the key makes a command replayed after a
   * reconnect safe to send again. A command sent while the socket is down
   * rejects immediately — the caller learns sooner, and a queue of writes
   * waiting on a connection is a queue nobody can reason about.
   */
  command(build: (id: string, idempotency: string) => Inbound): Promise<void> {
    const id = this.rooms.commandId();
    return new Promise((resolve, reject) => {
      this.awaiting.add(id, { resolve, reject });
      if (!this.socket.send(build(id, `${id}-${this.stamp}`))) {
        this.awaiting.settle({ type: 'error', id, error: 'not connected' });
      }
    });
  }

  /** Re-declares every room: a new socket is unknown to the server. */
  private rejoin(): void {
    this.rooms.reset();
    for (const room of this.rooms.rooms()) {
      this.socket.send({ type: 'join', id: this.rooms.joinId(), room });
    }
    if (this.rooms.settled()) this.announceReady();
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
    if (this.awaiting.settle(frame)) return;
    if (frame.type === 'ack' && frame.id && this.rooms.acknowledge(frame.id)) {
      this.announceReady();
    }
  }

  private announceReady(): void {
    this.listeners.notify((l) => l.onReady?.());
  }
}
