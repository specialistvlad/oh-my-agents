import type { ReactNode } from 'react';

import { Rail } from '@/components/workspace/Rail';
import { Splitter } from '@/components/workspace/Splitter';
import type { Layout, PanelID } from '@/workspace/layout';

type Props = {
  layout: Layout;
  left: ReactNode;
  centre: ReactNode;
  right: ReactNode;
  onPanel: (panel: PanelID) => void;
  onResizeLeft: (width: number) => void;
  onResizeRight: (width: number) => void;
};

/**
 * Three resizable columns behind an icon rail (ADR-0011).
 *
 * A collapsed column is not rendered at all rather than rendered at zero
 * width: an empty box still takes its borders and its focus order, and a
 * splitter beside nothing is a handle that resizes air.
 */
export function Workspace({
  layout,
  left,
  centre,
  right,
  onPanel,
  onResizeLeft,
  onResizeRight,
}: Props) {
  return (
    <div className="flex h-dvh overflow-hidden bg-background text-foreground">
      <Rail
        panel={layout.panel}
        collapsed={layout.left === 0}
        onSelect={onPanel}
      />

      {layout.left > 0 ? (
        <>
          <aside
            className="flex flex-col overflow-y-auto"
            style={{ width: layout.left }}
            aria-label="Objects">
            {left}
          </aside>
          <Splitter
            width={layout.left}
            side="left"
            label="Resize the objects panel"
            onResize={onResizeLeft}
          />
        </>
      ) : null}

      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        {centre}
      </main>

      {layout.right > 0 ? (
        <>
          <Splitter
            width={layout.right}
            side="right"
            label="Resize the inspector"
            onResize={onResizeRight}
          />
          <aside
            className="flex flex-col overflow-y-auto border-l border-border"
            style={{ width: layout.right }}
            aria-label="Inspector">
            {right}
          </aside>
        </>
      ) : null}
    </div>
  );
}
