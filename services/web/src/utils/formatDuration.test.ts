import { describe, expect, it } from 'vitest';

import { formatDuration } from './formatDuration';

describe('formatDuration', () => {
  it('renders sub-second values in milliseconds', () => {
    expect(formatDuration(0)).toBe('0ms');
    expect(formatDuration(999)).toBe('999ms');
  });

  it('drops to the two largest units', () => {
    expect(formatDuration(1_000)).toBe('1s');
    expect(formatDuration(65_000)).toBe('1m 5s');
    expect(formatDuration(3_725_000)).toBe('1h 2m');
  });

  it('returns a placeholder for values that are not durations', () => {
    expect(formatDuration(-1)).toBe('—');
    expect(formatDuration(Number.NaN)).toBe('—');
  });
});
