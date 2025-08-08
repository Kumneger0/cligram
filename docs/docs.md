## Cligram User Guide

Make Telegram quick, quiet, and keyboard-first. This guide covers the layout, shortcuts, actions, search, attachments, and configuration—kept light and practical.

---

## Quick Start

- Login: run `cligram login` and follow the prompts.
- Logout: run `cligram logout`.
- Upgrade: on Debian/Ubuntu/Alpine, `cligram upgrade` updates to the latest release.

---

## Layout

- Sidebar: your chats list (Users, Groups, Channels).
- Chat area: messages of the selected chat.
- Focus is shown with a green border. Use Tab to move focus.

---

## Navigation & Shortcuts

- Focus
  - Tab: toggle focus between the sidebar and the chat area.
- Move
  - ↑ / k: move up (in chat list or messages)
  - ↓ / j: move down (in chat list or messages)
- Filter (sidebar)
  - c: show Channels
  - g: show Groups
  - u: show Users
- Search
  - ctrl + k: open the Search panel
  - Type to search. Results appear below the input
  - Tab switches focus between the input and the results list
  - The currently focused area has a double border
  - Enter opens the selected chat or item; Esc closes search

---

## Working in Chats

- Selecting messages: move with ↑/↓ (or k/j) inside the chat area to highlight a message.
- Actions on the selected message
  - d: delete
  - r: reply
  - e: edit
  - f: forward
  - u: open a direct message with the sender (from a group)

Tips
- Replies show the quoted preview above your draft while composing.
- Forward preserves original sender and timestamp context.

---

## Compose & Attachments

- ctrl + a: toggle the file picker when the input is focused
- After picking a file: optionally type a caption, then press Enter to send
- While sending: an upload progress indicator is shown

---

## Reading Behavior

- Typing indicator: if enabled, others see "typing…" while you compose
- Read receipts depend on your `readReceiptMode` (see Configuration)
  - default: only mark read when you interact (e.g., reply)
  - instant: mark read as soon as you view messages
  - never: never mark read automatically

---

## Configuration

The config file lives at `~/.cligram/user.config.json`. All fields are optional—anything you omit uses a sensible default.

Example

```json
{
  "chat": {
    "sendTypingState": true,
    "readReceiptMode": "default"
  },
  "privacy": {
    "lastSeenVisibility": "everyone"
  },
  "notifications": {
    "enabled": true,
    "showMessagePreview": true
  }
}
```

Options

- Chat
  - sendTypingState (boolean)
    - true: show "typing…" to others (default)
    - false: never show typing
  - readReceiptMode (string)
    - "default": mark read only when you interact (e.g., reply)
    - "instant": mark read as soon as you view messages
    - "never": never mark read automatically
- Privacy
  - lastSeenVisibility (string)
    - "everyone": anyone can see your last seen time
    - "contacts": only contacts can see it
    - "nobody": no one can see it
    - not set: uses your Telegram privacy settings
- Notifications
  - enabled (boolean)
    - true: show notifications (default)
    - false: disable all notifications
  - showMessagePreview (boolean)
    - true: include message content in notifications (default)
    - false: show only the sender name

Notes
- Changes take effect the next time you start the app.
- You can begin with an empty file and add only what you need.

---

## Troubleshooting

- File picker does not open: make sure the message input is focused, then press `ctrl + a` again. Update to the latest version if the issue persists.
- Messages not marked read: set `readReceiptMode` to `instant`.
- Others cannot see typing: set `sendTypingState` to `true`.
- Where is data stored: app data and session live under `~/.cligram/`.

---

Stay keyboard-first, and enjoy the silence. All defaults are safe—you can refine behavior any time via `~/.cligram/user.config.json`.