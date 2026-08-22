// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';

import { UNKNOWN, loadIdentity, saveIdentity } from './identity';

afterEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('identity', () => {
  it('round trips a name', () => {
    saveIdentity('vk');
    expect(loadIdentity()).toEqual({ kind: 'human', id: 'vk' });
  });

  // The feed is only worth reading if something declares a name, so an
  // unnamed person still gets attribution rather than an empty actor.
  it('falls back rather than declaring nobody', () => {
    expect(loadIdentity()).toEqual(UNKNOWN);
    expect(UNKNOWN.id).not.toBe('');
    expect(saveIdentity('   ')).toEqual(UNKNOWN);
  });

  it('trims what it is given', () => {
    expect(saveIdentity('  vk  ')).toEqual({ kind: 'human', id: 'vk' });
    expect(loadIdentity().id).toBe('vk');
  });

  it('clears back to the fallback', () => {
    saveIdentity('vk');
    saveIdentity('');
    expect(loadIdentity()).toEqual(UNKNOWN);
  });

  it('survives a browser that refuses storage', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied');
    });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('denied');
    });
    expect(loadIdentity()).toEqual(UNKNOWN);
    expect(() => saveIdentity('vk')).not.toThrow();
  });
});
