import { describe, expect, it } from 'vitest';

import { backoffDelay } from './backoff';

describe('backoffDelay', () => {
  it('grows exponentially', () => {
    const mid = () => 0.5;
    expect(backoffDelay(1, mid)).toBeLessThan(backoffDelay(2, mid));
    expect(backoffDelay(2, mid)).toBeLessThan(backoffDelay(3, mid));
  });

  it('stops growing at the ceiling', () => {
    const mid = () => 0.5;
    expect(backoffDelay(20, mid)).toBe(backoffDelay(30, mid));
    expect(backoffDelay(30, mid)).toBeLessThanOrEqual(15_000);
  });

  // Without jitter, every client dropped by one restart returns at the same
  // instant and knocks the server over again.
  it('spreads attempts out', () => {
    expect(backoffDelay(5, () => 0)).not.toBe(backoffDelay(5, () => 1));
  });

  it('never returns a negative or zero delay', () => {
    for (let attempt = 1; attempt <= 20; attempt++) {
      expect(backoffDelay(attempt, () => 0)).toBeGreaterThan(0);
    }
  });
});
