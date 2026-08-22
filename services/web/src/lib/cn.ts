import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * Joins class names, letting a later utility win over an earlier one.
 *
 * Plain concatenation cannot do that: `px-2 px-4` leaves both in the class
 * list and the winner is whichever CSS rule happens to come last, not the
 * one the caller passed last. twMerge resolves it by intent, which is what
 * makes a component's classes overridable from outside.
 */
export function cn(...classes: ClassValue[]): string {
  return twMerge(clsx(classes));
}
