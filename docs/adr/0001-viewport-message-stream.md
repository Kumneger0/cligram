# 1. Viewport-Based Message Stream

## Context

The chat view previously used Charm's `bubbles/list.Model` with a hardcoded item height of 1 (`MessagesDelegate.Height() == 1`). However, Telegram messages have variable heights due to text wrapping, quotes, reactions, link previews, and metadata. This caused severe rendering artifacts, cursor-locking hacks (`m.ChatUI.Select(len-1)` on every frame render), and broken scrolling.

## Decision

We replace `bubbles/list.Model` in the chat area with `viewport.Model`, managing a `selectedMessageIndex` directly over an in-memory message buffer (`[]types.FormattedMessage`).

## Consequences

- Messages can have arbitrary multiline heights, reply banners, and reaction badges without breaking terminal scrolling.
- Navigation (`j`/`k`, `↑`/`↓`) steps through message blocks and scrolls them into view dynamically.
- The sidebar continues to use `bubbles/list.Model` where item heights are uniformly fixed.
