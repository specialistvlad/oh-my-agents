/**
 * The rooms a client wants and the joins it is still waiting on.
 *
 * Separate from the socket because the two have different lifetimes: rooms
 * are declared once and outlive every connection, while outstanding joins
 * belong to the socket that sent them and are void the moment it drops.
 */
export class Membership {
  private readonly wanted = new Set<string>();
  private readonly outstanding = new Set<string>();
  private nextId = 0;

  add(room: string): void {
    this.wanted.add(room);
  }

  remove(room: string): void {
    this.wanted.delete(room);
  }

  /** Every room to (re-)join on a new connection. */
  rooms(): string[] {
    return [...this.wanted];
  }

  /** A correlation id for a command whose reply nobody is waiting on. */
  commandId(): string {
    this.nextId += 1;
    return `c${this.nextId}`;
  }

  /** A correlation id for a join, recorded until it is acknowledged. */
  joinId(): string {
    const id = this.commandId();
    this.outstanding.add(id);
    return id;
  }

  /** Forgets outstanding joins, because a new socket has to re-send them. */
  reset(): void {
    this.outstanding.clear();
  }

  /**
   * Records an acknowledgement. Reports true when the last one lands, which
   * is when every room is joined and it is safe to fetch state.
   */
  acknowledge(id: string): boolean {
    return this.outstanding.delete(id) && this.outstanding.size === 0;
  }

  /** True when there is nothing to wait for. */
  settled(): boolean {
    return this.wanted.size === 0;
  }
}
