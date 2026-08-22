import { describe, expect, it } from 'vitest';

import { formatValue } from './format';
import type { Value } from './types';

const v = (kind: Value['kind'], value: unknown): Value => ({ kind, value });

describe('formatValue', () => {
  it('reads every kind', () => {
    expect(formatValue(v('text', 'hello'))).toBe('hello');
    expect(formatValue(v('number', 4.5))).toBe('4.5');
    expect(formatValue(v('bool', true))).toBe('yes');
    expect(formatValue(v('bool', false))).toBe('no');
    expect(formatValue(v('duration', '1m30s'))).toBe('1m30s');
    expect(formatValue(v('select', 'high-5n8p'))).toBe('high-5n8p');
    expect(formatValue(v('multi_select', ['ui-7q1s', 'api-3t5u']))).toBe(
      'ui-7q1s, api-3t5u'
    );
    expect(formatValue(v('url', 'https://example.test'))).toBe(
      'https://example.test'
    );
  });

  it('distinguishes an agent from a person', () => {
    expect(formatValue(v('actor', { kind: 'agent', id: 'builder-1' }))).toBe(
      'builder-1 (agent)'
    );
    expect(formatValue(v('actor', { kind: 'human', id: 'vk' }))).toBe('vk');
  });

  it('shows a placeholder for an absent value', () => {
    expect(formatValue(undefined)).toBe('—');
    expect(formatValue(v('text', ''))).toBe('—');
    expect(formatValue(v('multi_select', []))).toBe('—');
  });

  // The api refuses to store a value whose payload contradicts its kind, so
  // seeing one means something upstream is wrong. Coercing it would hide that.
  it('does not coerce a payload that contradicts its kind', () => {
    expect(formatValue(v('number', 'not a number'))).toBe('?');
    expect(formatValue(v('bool', 7))).toBe('?');
    expect(formatValue(v('date', 'the third of never'))).toBe('?');
    expect(formatValue(v('multi_select', 'not a list'))).toBe('?');
    expect(formatValue(v('actor', 'just a name'))).toBe('?');
  });

  it('shows a date in the reader locale', () => {
    const shown = formatValue(v('date', '2026-08-21T12:00:00Z'));
    expect(shown).not.toBe('?');
    expect(shown).toContain('2026');
  });
});
