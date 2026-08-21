import { describe, expect, it } from 'vitest';

import { socketUrl } from './client';

describe('socketUrl', () => {
  it('swaps the scheme and appends the endpoint', () => {
    expect(socketUrl('http://localhost:39170')).toBe('ws://localhost:39170/ws');
    expect(socketUrl('https://oma.example')).toBe('wss://oma.example/ws');
  });

  it('leaves the host and port alone', () => {
    expect(socketUrl('http://127.0.0.1:39170')).toBe('ws://127.0.0.1:39170/ws');
  });
});
