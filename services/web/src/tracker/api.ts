import type { Item, Schema } from './types';

/** The tracker's read surface for one project. */
export class TrackerAPI {
  constructor(
    private readonly apiUrl: string,
    private readonly project: string
  ) {}

  schema(signal?: AbortSignal): Promise<Schema> {
    return this.get<Schema>('schema', signal);
  }

  async items(signal?: AbortSignal): Promise<Item[]> {
    const page = await this.get<{ items?: Item[] }>(
      'items?sort=created_at',
      signal
    );
    return page.items ?? [];
  }

  /**
   * One item, or null if it is gone.
   *
   * A 404 is an answer rather than a failure here: an event and a fetch race
   * each other, so hearing about an item that has since been deleted is
   * ordinary rather than exceptional.
   */
  async item(id: string, signal?: AbortSignal): Promise<Item | null> {
    const response = await fetch(this.url(`items/${id}`), { signal });
    if (response.status === 404) return null;
    if (!response.ok)
      throw new Error(`${response.status} ${response.statusText}`);
    return (await response.json()) as Item;
  }

  private async get<T>(path: string, signal?: AbortSignal): Promise<T> {
    const response = await fetch(this.url(path), { signal });
    if (!response.ok)
      throw new Error(`${response.status} ${response.statusText}`);
    return (await response.json()) as T;
  }

  private url(path: string): string {
    return `${this.apiUrl}/projects/${this.project}/tracker/${path}`;
  }
}
