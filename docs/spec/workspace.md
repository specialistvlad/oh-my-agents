# Spec: the workspace shell

Implementation spec for `services/web/src/workspace` and
`services/web/src/components/workspace`. The decision is
[ADR-0011](../adr/0011-three-column-workspace.md).

Status: **implemented.**

## Shape

```
┌──┬──────────────┬───────────────────────┬─────────────┐
│ ▣│  objects     │ [tab] [tab] [tab]     │  inspector  │
│ ▤│   project    │                       │   title     │
│  │    └ item    │   the open thing      │   status    │
│  │    └ item    │                       │   moves     │
└──┴──────────────┴───────────────────────┴─────────────┘
  rail   left            centre                right
```

The rail selects which panel the left column shows, and is what brings a
collapsed column back — which is why collapsing to zero is safe. Choosing the
panel already showing collapses it, the way an IDE sidebar behaves.

## Selection and the active tab are different

A single click in the tree **selects**, filling the inspector without opening
anything. A double click **opens** a tab. That separation is ADR-0011's
deliberate cost: it lets you look at one thing while working on another, and it
means the UI has two "current" things to show distinctly.

## Tombstones

An item deleted by somebody else does not take its tab with it. The tab stays,
struck through and titled "deleted", until it is closed deliberately. A
selection whose object is gone reports that rather than emptying.

Vanishing would look like a bug in the application, lose whatever was on
screen, and slide every neighbouring tab under the cursor.

`useTombstones` waits for the item list to have loaded before marking anything,
because an empty list before the first fetch is not evidence that anything was
deleted.

## Resizing

Hand-rolled: a pointer-move handler and a clamp (`clampWidth`). Pointer capture
is what makes a drag survive the cursor leaving the handle.

Handles carry `role="separator"` with `aria-valuenow`, are focusable, and
respond to arrow keys, `Home` and `End`. Keyboard resizing is the part that
gets skipped, and skipping it puts the column widths out of reach of anyone not
using a mouse.

Dragged below 120px a column collapses rather than clamping at its 180px
minimum: a column that refuses to shrink and springs back reads as broken. No
column may exceed 40% of the window — the centre is what the app is for.

## Layout is per device

Widths and the active panel go to `localStorage`, read after mount rather than
during render. A laptop and a wide monitor want different widths, so syncing
them would make both worse, and every drag would become a write.

Anything unreadable falls back to the default: a corrupt preference is not
worth a blank page, and a browser refusing storage entirely is not a reason to
stop working.

## Known limits

- The rail's second panel (activity) is a placeholder with nothing behind it.
- The centre shows an item's title and body; editing them is not built.
- Comments and links have api surface but no UI.
