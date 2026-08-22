/**
 * A WebSocket stand-in for tests.
 *
 * The client's reconnect, re-join and acknowledgement handling are the parts
 * most likely to break and the parts a real socket makes hardest to test, so
 * they are driven from here instead.
 */
export class FakeSocket {
  static readonly OPEN = 1;
  static instances: FakeSocket[] = [];

  static reset(): void {
    FakeSocket.instances = [];
  }

  /** The socket most recently constructed. */
  static last(): FakeSocket {
    const socket = FakeSocket.instances.at(-1);
    if (!socket) throw new Error('no socket was constructed');
    return socket;
  }

  readyState = 0;
  readonly sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(readonly url: string) {
    FakeSocket.instances.push(this);
  }

  send(data: string): void {
    this.sent.push(data);
  }

  close(): void {
    if (this.readyState === 3) return;
    this.readyState = 3;
    this.onclose?.();
  }

  /** Completes the handshake. */
  accept(): void {
    this.readyState = FakeSocket.OPEN;
    this.onopen?.();
  }

  /** Delivers a server frame. */
  deliver(frame: unknown): void {
    this.onmessage?.({ data: JSON.stringify(frame) });
  }

  /** What the client sent, decoded. */
  frames(): Array<Record<string, unknown>> {
    return this.sent.map((raw) => JSON.parse(raw) as Record<string, unknown>);
  }
}
