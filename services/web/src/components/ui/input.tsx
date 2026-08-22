import type { InputHTMLAttributes } from 'react';

import { cn } from '@/lib/cn';

export function Input({
  className,
  ...rest
}: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        'h-9 w-full rounded-app border border-border bg-surface px-3 text-sm',
        'outline-none placeholder:text-muted-foreground',
        'focus-visible:ring-2 focus-visible:ring-primary/50',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className
      )}
      {...rest}
    />
  );
}
