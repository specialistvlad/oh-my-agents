import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RealtimeClient } from './client';
import { FakeSocket } from './fakeSocket';
import type { RealtimeEvent, Status } from './types';
import { socketUrl } from './url';

describe('socketUrl', () => {
  it('swaps the scheme and appends the endpoint', () => {
    expect(socketUrl('http://localhost:39170')).toBe('ws://localhost:39170/ws');
    expect(socketUrl('https://oma.example')).toBe('wss://oma.example/ws');
  });
});

describe('RealtimeClient', () => {
  let original: unknown;

  beforeEach(() => {
    original = globalThis.WebSocket;
    FakeSocket.reset();
    (globalThis as { WebSocket: unknown }).WebSocket = FakeSocket;
  });

  afterEach(() => {
    (globalThis as { WebSocket: unknown }).WebSocket = original;
    vi.useRealTimers();
  });

  const connected = () => {
    const client = new RealtimeClient('ws://test/ws');
    client.start();
    FakeSocket.last().accept();
    return client;
  };

  it('reports its status as the socket moves', () => {
    const seen: Status[] = [];
    const client = new RealtimeClient('ws://test/ws');
    client.listen({ onStatus: (s) => seen.push(s) });
    client.start();
    expect(seen).toContain('connecting');
    FakeSocket.last().accept();
    expect(seen).toContain('open');
  });

  // Rooms are declared before the socket exists; nothing may be dropped for
  // arriving early.
  it('sends a join requested before the socket opened', () => {
    const client = new RealtimeClient('ws://test/ws');
    client.start();
    client.join('settings');
    expect(FakeSocket.last().frames()).toHaveLength(0);

    FakeSocket.last().accept();
    expect(FakeSocket.last().frames()).toEqual([
      expect.objectContaining({ type: 'join', room: 'settings' }),
    ]);
  });

  // Ready is what tells a caller it may fetch. Firing it before the server
  // acknowledged the join would reopen the window join-first exists to close.
  it('signals ready only once every join is acknowledged', () => {
    const ready = vi.fn();
    const client = new RealtimeClient('ws://test/ws');
    client.listen({ onReady: ready });
    client.start();
    client.join('a');
    client.join('b');
    FakeSocket.last().accept();

    const joins = FakeSocket.last().frames();
    FakeSocket.last().deliver({ type: 'ack', id: joins[0].id });
    expect(ready).not.toHaveBeenCalled();

    FakeSocket.last().deliver({ type: 'ack', id: joins[1].id });
    expect(ready).toHaveBeenCalledTimes(1);
  });

  it('reports events and resyncs', () => {
    const events: RealtimeEvent[] = [];
    const resyncs: string[] = [];
    const client = connected();
    client.listen({
      onEvent: (e) => events.push(e),
      onResync: (room) => resyncs.push(room),
    });

    FakeSocket.last().deliver({
      type: 'event',
      room: 'settings',
      seq: 7,
      kind: 'setting.changed',
      data: { key: 'a' },
    });
    expect(events).toEqual([
      { room: 'settings', seq: 7, kind: 'setting.changed', data: { key: 'a' } },
    ]);

    FakeSocket.last().deliver({ type: 'resync', room: 'settings' });
    expect(resyncs).toEqual(['settings']);
  });

  it('ignores frames it cannot parse or act on', () => {
    const events: RealtimeEvent[] = [];
    const client = connected();
    client.listen({ onEvent: (e) => events.push(e) });

    FakeSocket.last().onmessage?.({ data: 'not json' });
    FakeSocket.last().deliver({ type: 'event' }); // no room or kind
    expect(events).toEqual([]);
  });

  // The reconnect that matters: the server forgets everything, so a new
  // socket has to re-declare the rooms or the client goes quiet forever.
  it('reconnects and re-joins its rooms', () => {
    vi.useFakeTimers();
    const client = connected();
    client.join('settings');
    const first = FakeSocket.last();
    expect(FakeSocket.instances).toHaveLength(1);

    first.close();
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.instances.length).toBeGreaterThan(1);

    const second = FakeSocket.last();
    second.accept();
    expect(second.frames()).toEqual([
      expect.objectContaining({ type: 'join', room: 'settings' }),
    ]);
  });

  // A remount stops and restarts the client. The first socket's close event
  // arrives after that, and reconnecting from it leaves two live sockets
  // delivering every event twice.
  it('ignores a close from a socket it already discarded', () => {
    vi.useFakeTimers();
    const client = connected();
    const first = FakeSocket.last();

    client.stop();
    client.start();
    expect(FakeSocket.instances).toHaveLength(2);

    first.onclose?.();
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.instances).toHaveLength(2);
  });

  it('delivers each event once after a remount', () => {
    const events: RealtimeEvent[] = [];
    const client = connected();
    client.listen({ onEvent: (e) => events.push(e) });
    client.join('settings');

    const first = FakeSocket.last();
    client.stop();
    client.start();
    FakeSocket.last().accept();
    first.onclose?.();

    FakeSocket.last().deliver({
      type: 'event',
      room: 'settings',
      seq: 1,
      kind: 'k',
    });
    expect(events).toHaveLength(1);
  });

  it('does not reconnect after stop', () => {
    vi.useFakeTimers();
    const client = connected();
    client.stop();
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.instances).toHaveLength(1);
  });

  it('stops delivering to a removed listener', () => {
    const events: RealtimeEvent[] = [];
    const client = connected();
    const stop = client.listen({ onEvent: (e) => events.push(e) });
    stop();

    FakeSocket.last().deliver({ type: 'event', room: 'r', seq: 1, kind: 'k' });
    expect(events).toEqual([]);
  });
});
