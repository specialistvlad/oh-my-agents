import type { Listener } from './types';

/**
 * The set of things watching a connection.
 *
 * Separated from the client because registration and notification are their
 * own concern, and keeping them here leaves the client to be about sockets.
 */
export class Listeners {
  private readonly all = new Set<Listener>();

  /** Registers a listener and returns the function that removes it. */
  add(listener: Listener): () => void {
    this.all.add(listener);
    return () => this.all.delete(listener);
  }

  /** Calls one handler on every listener that has it. */
  notify(call: (listener: Listener) => void): void {
    this.all.forEach(call);
  }
}
