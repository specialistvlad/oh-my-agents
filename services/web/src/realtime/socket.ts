import { backoffDelay } from './backoff';
import type { Status } from './types';

export type SocketHandlers = {
  /** The socket is usable. Everything a connection needs re-declared goes here. */
  onOpen: () => void;
  onFrame: (raw: unknown) => void;
  /** The socket went away. Anything waiting on it can no longer be answered. */
  onDrop: () => void;
  onStatus: (status: Status) => void;
};

/**
 * One connection's lifecycle: opening, dropping, reconnecting.
 *
 * Separated from the client because this is the part with nothing to do with
 * rooms or commands, and the part where the subtle failures live — a socket
 * acting after it was abandoned, an error path opening a second one, a
 * reconnect racing a stop.
 */
export class Socket {
  private socket: WebSocket | null = null;
  private attempt = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = false;
  private status: Status = 'closed';

  constructor(
    private readonly url: string,
    private readonly handlers: SocketHandlers
  ) {}

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

    const socket = this.socket;
    this.socket = null;
    if (socket) {
      detach(socket);
      socket.close();
    }
    this.handlers.onDrop();
    this.setStatus('closed');
  }

  /** Sends a frame, reporting whether there was a connection to send it on. */
  send(frame: unknown): boolean {
    if (this.socket?.readyState !== WebSocket.OPEN) return false;
    this.socket.send(JSON.stringify(frame));
    return true;
  }

  private open(): void {
    this.setStatus(this.attempt === 0 ? 'connecting' : 'reconnecting');
    const socket: WebSocket = new WebSocket(this.url);
    this.socket = socket;

    socket.onopen = () => {
      this.attempt = 0;
      this.setStatus('open');
      this.handlers.onOpen();
    };
    socket.onmessage = (message: MessageEvent) =>
      this.handlers.onFrame(message.data);
    socket.onclose = () => {
      // Only the socket we are actually using may trigger a reconnect. A
      // close arrives asynchronously, so one from a socket already replaced
      // or stopped can land after the client has moved on — and reconnecting
      // from that leaves two live sockets delivering every event twice.
      if (this.socket !== socket) return;
      this.socket = null;
      this.handlers.onDrop();
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

  private setStatus(status: Status): void {
    if (status === this.status) return;
    this.status = status;
    this.handlers.onStatus(status);
  }
}

/**
 * Removes every handler before a socket is abandoned.
 *
 * A socket that still has handlers attached can act on behalf of a client
 * that has already moved on. Detaching first is what makes abandoning final.
 */
function detach(socket: WebSocket): void {
  socket.onopen = null;
  socket.onmessage = null;
  socket.onclose = null;
  socket.onerror = null;
}
