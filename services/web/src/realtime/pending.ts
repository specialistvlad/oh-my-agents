import type { Outbound } from './types';

type Waiter = { resolve: () => void; reject: (reason: Error) => void };

/**
 * Commands sent and not yet answered.
 *
 * A bidirectional socket has no request/response pairing of its own, so this
 * is what turns a stream of frames back into something a caller can await:
 * every command carries an id, and its reply carries the same one.
 */
export class Pending {
  private readonly waiting = new Map<string, Waiter>();

  /** Records a command as outstanding. */
  add(id: string, waiter: Waiter): void {
    this.waiting.set(id, waiter);
  }

  /**
   * Settles the command a frame answers, reporting whether it answered one.
   * A frame nobody is waiting on is somebody else's — an event, a resync —
   * and must fall through.
   */
  settle(frame: Outbound): boolean {
    if (!frame.id) return false;
    const waiter = this.waiting.get(frame.id);
    if (!waiter) return false;

    this.waiting.delete(frame.id);
    if (frame.type === 'error') {
      waiter.reject(new Error(frame.error ?? 'command failed'));
    } else {
      waiter.resolve();
    }
    return true;
  }

  /**
   * Fails everything outstanding, because the socket that would have
   * answered is gone.
   *
   * Leaving them pending would hang a caller until the page closes. The
   * honest answer is that nobody knows whether they were applied — which is
   * exactly what the idempotency key on the retry is for.
   */
  abandon(): void {
    const waiting = [...this.waiting.values()];
    this.waiting.clear();
    waiting.forEach((waiter) => waiter.reject(new Error('connection lost')));
  }
}
