# 2. Live Modal Compositor

## Context

Modals (such as Search `Ctrl+K`, Forward `f`, and Stories) were previously wrapped in a static `overlay.New` struct initialized at program startup. Because `Manager.Overlay` was never re-bound to updated background state, opening any modal composited the dialog over a frozen snapshot of the startup UI.

## Decision

We eliminate static overlay wrapper models in favor of dynamic frame-by-frame compositing. In `Manager.View()`, when in `ModalView`, the modal view is composited directly onto the current frame of `m.Background.View()` using `overlay.Composite(...)`.

## Consequences

- Modals always render on top of the live, current chat view and active background state.
- Modal state management is decoupled from background state updates.
