import type { Value } from './types';

/**
 * A field's value as a person should read it.
 *
 * Formatting lives apart from the components that show it so it can be tested
 * without rendering anything, and so a value looks the same wherever it
 * appears — the inspector, a card, a list.
 *
 * A value whose payload does not match its kind is shown as a placeholder
 * rather than crashing or coercing. The api refuses to store one, so seeing it
 * means something upstream is wrong and hiding that helps nobody.
 */
export function formatValue(value: Value | undefined): string {
  if (!value) return '—';
  switch (value.kind) {
    case 'text':
    case 'markdown':
    case 'url':
    case 'select':
      return asString(value.value);
    case 'number':
      return typeof value.value === 'number' ? String(value.value) : '?';
    case 'bool':
      if (typeof value.value !== 'boolean') return '?';
      return value.value ? 'yes' : 'no';
    case 'date':
      return formatDate(value.value);
    case 'duration':
      return asString(value.value);
    case 'multi_select':
      return Array.isArray(value.value) ? value.value.join(', ') || '—' : '?';
    case 'actor':
      return formatActor(value.value);
    case 'item':
      return asString(value.value);
    default:
      return '?';
  }
}

function asString(raw: unknown): string {
  return typeof raw === 'string' ? raw || '—' : '?';
}

/** Dates arrive as RFC 3339 and are shown in the reader's own locale. */
function formatDate(raw: unknown): string {
  if (typeof raw !== 'string') return '?';
  const at = new Date(raw);
  return Number.isNaN(at.getTime()) ? '?' : at.toLocaleDateString();
}

function formatActor(raw: unknown): string {
  if (!raw || typeof raw !== 'object') return '?';
  const { kind, id } = raw as { kind?: string; id?: string };
  if (!id) return '—';
  // An agent is worth distinguishing from a person at a glance; both are
  // self-declared and neither is verified (ADR-0012).
  return kind === 'agent' ? `${id} (agent)` : id;
}
