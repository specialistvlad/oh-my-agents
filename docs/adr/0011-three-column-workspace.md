# ADR-0011: The UI is a three-column workspace built around state that arrives

- Status: Accepted
- Date: 2026-08-21
- Scope: `services/web`
- Relates to: ADR-0004, ADR-0008, ADR-0010

## Context

The web app is one page with a list on it. What it has to become is a place to
work: many objects, deeply nested (ADR-0004 puts no bound on depth), several
open at once, and other people and agents changing them while you look.

That last part is the constraint that shapes everything else. Every other
decision here could be made by looking at an IDE; this one cannot, because an
IDE's file does not get renamed by someone else mid-sentence.

## Decision

### Three columns, each resizable

    ┌──┬──────────────┬───────────────────────┬─────────────┐
    │ ▣│  objects     │ [story] [bug] [epic]  │  inspector  │
    │ ▤│   epic       │                       │   name      │
    │ ▥│    └ story   │   the open thing      │   status    │
    │  │    └ bug     │                       │   fields    │
    └──┴──────────────┴───────────────────────┴─────────────┘
      rail   left            centre                right

A narrow icon rail selects which panel the left column shows. Objects is the
first and the default; the rail exists so later panels — search, activity,
agents — arrive without another decision.

### The centre holds several open tabs

An agent working on one item while a person reads another is the normal case
here, not an advanced one, so one-at-a-time would be wrong from the first day.

### The inspector follows selection, not the active tab

Selecting in the tree shows a thing in the inspector without opening it. That
means two "current" things — what is selected and what is open — and the UI has
to show both distinctly, or people will not be able to predict what the
inspector is describing. That legibility is the price of being able to look at
one thing while working on another, and it is worth it.

### Tabs and selections become tombstones, never vanish

An object you have open can be deleted by somebody else. When that happens the
tab stays and says so.

A tab disappearing mid-sentence is worse than a tab that says the thing is
gone: the first looks like a bug in the application, loses whatever was on
screen, and silently moves every tab beside it under the cursor. The same holds
for a selection whose object is removed — the inspector reports it rather than
emptying.

This is the rule the whole UI is shaped by: **things arrive and depart while
you are looking at them**, and every component has to have an answer for that
rather than assuming what it rendered still exists.

### Nothing in a component fetches

Components render from a store fed by the socket (ADR-0008). The one fetch is
on connect and after a resync, and it belongs to the store, not to a view.

A component that fetches is a component that polls the moment it is rendered
twice, and it is the single easiest way to lose the property this system is
built for.

### Layout lives per device, in the browser

Column widths, the active rail panel and open tabs go to `localStorage`. A
laptop and a wide monitor want different widths, so syncing them makes both
worse, and every drag would become a write.

This is the one place where sync being first-class does not apply, and it is
deliberate: the layout is a property of the screen, not of the work.

### Resizing is hand-rolled, and keyboard operable

A splitter is a pointer-move handler and a clamp. Taking a dependency for that
is how a dependency list becomes nine unused packages, which this project has
already done once.

Handles are focusable, respond to arrow keys, and carry the separator role with
its value. Keyboard resizing is the part that gets skipped, and skipping it
makes the column widths unreachable for anyone not using a mouse. Columns have
minimum widths and collapse to the rail rather than to zero, so a column can
always be recovered.

## Consequences

**Two "current" things is a real cost.** Selection and the active tab can point
at different objects, and a person who cannot tell which the inspector is
describing will mistrust it. Showing both distinctly is not decoration.

**Every list and tree must tolerate its contents changing underneath.** Nothing
may hold an index, cache a lookup by position, or assume that what it rendered
last frame still exists. This is where the bugs will be, and they will look
like rare glitches rather than logic errors.

**Layout is invisible to the server**, so it cannot be inspected when someone
reports the app looking wrong, and it does not follow them to another machine.
Both are accepted; the fix if it ever matters is to sync it deliberately, not
to have stored it centrally by accident.

**The 150-line component cap will bite**, because a resizable three-column
shell is genuinely more than 150 lines of markup and handlers. It splits along
its own seams — a column, a rail, a tab strip, a handle, and a hook holding the
sizes — and that split is wanted anyway. Vendored components under
`components/ui` are already exempt (ADR-0003 amendment in eslint config).

**Tombstones need a lifetime.** A tab for a deleted object cannot stay forever,
and this ADR does not decide when it goes. Closing it on the next navigation is
the likely answer, but it is left open rather than guessed.
