import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RealtimeClient } from './client';
import { deleteSetting, setSetting } from './commands';
import { FakeSocket } from './fakeSocket';

describe('commands', () => {
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

  const open = () => {
    const client = new RealtimeClient('ws://test/ws');
    client.start();
    FakeSocket.last().accept();
    return client;
  };

  it('resolves when the reply arrives', async () => {
    const client = open();
    const done = setSetting(client, 'agent/model', { m: 'opus' });

    const sent = FakeSocket.last().frames()[0];
    expect(sent).toMatchObject({ type: 'set', key: 'agent/model' });
    expect(sent.idempotency).toBeTruthy();

    FakeSocket.last().deliver({ type: 'ack', id: sent.id });
    await expect(done).resolves.toBeUndefined();
  });

  it('rejects when the server refuses', async () => {
    const client = open();
    const done = deleteSetting(client, 'gone');
    const sent = FakeSocket.last().frames()[0];

    FakeSocket.last().deliver({
      type: 'error',
      id: sent.id,
      error: 'not found',
    });
    await expect(done).rejects.toThrow('not found');
  });

  // A caller learns immediately rather than waiting on a queue nobody can
  // reason about.
  it('rejects while disconnected', async () => {
    const client = new RealtimeClient('ws://test/ws');
    client.start();
    await expect(setSetting(client, 'k', {})).rejects.toThrow('not connected');
  });

  // Nobody knows whether an in-flight command was applied, so it must fail
  // rather than hang forever.
  it('fails commands in flight when the socket drops', async () => {
    const client = open();
    const done = setSetting(client, 'k', {});
    FakeSocket.last().close();
    await expect(done).rejects.toThrow('connection lost');
  });

  // Two commands in flight is the case correlation ids exist for.
  it('matches each reply to its own command', async () => {
    const client = open();
    const first = setSetting(client, 'a', {});
    const second = setSetting(client, 'b', {});
    const [sentA, sentB] = FakeSocket.last().frames();

    FakeSocket.last().deliver({
      type: 'error',
      id: sentB.id,
      error: 'b failed',
    });
    FakeSocket.last().deliver({ type: 'ack', id: sentA.id });

    await expect(first).resolves.toBeUndefined();
    await expect(second).rejects.toThrow('b failed');
  });

  // A command reply must not be mistaken for a room acknowledgement.
  it('does not treat a command ack as a join', async () => {
    const ready = vi.fn();
    const client = open();
    client.listen({ onReady: ready });
    client.join('room');

    const [join, set] = (() => {
      const done = setSetting(client, 'k', {});
      void done.catch(() => undefined);
      return FakeSocket.last().frames();
    })();

    FakeSocket.last().deliver({ type: 'ack', id: set.id });
    expect(ready).not.toHaveBeenCalled();

    FakeSocket.last().deliver({ type: 'ack', id: join.id });
    expect(ready).toHaveBeenCalledTimes(1);
  });
});
