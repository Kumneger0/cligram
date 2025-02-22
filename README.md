# Telegram CLI

A Telegram CLI client built with TypeScript and Node.js.

> **Warning:** This project is still under development and may not be ready for production use.

## Usage

This command-line interface (CLI) application offers an interactive environment with three primary components:

1. **Sidebar**: Displays a list of available chats.
2. **Chat Area**: Shows the content of the selected chat.
3. **Input Field**: Allows users to type and send messages.

Users can navigate between these components, with the currently focused component highlighted by a border.

## Navigation

- **Switching Focus**: Press the **Tab** key to cycle focus between the Sidebar, Chat Area, and Input Field. The focused component is indicated by a border.

### Sidebar Navigation
Navigate through your chat list using:
- **↑** or **k** - Move up
- **↓** or **j** - Move down

The selected chat will be highlighted as you navigate through the list.
## Chat Area Navigation

- **Moving Through Messages:**
  - Press the - **↑** key or **k** to move to an earlier message.
  - Press the - **↓** Down Arrow** key or **j** to move to a later message.

The currently selected message is highlighted with a **blue background** and **white text**, so you can easily see which message is active.


## User Interaction

- **Input Field**: Focus on the Input Field to type your message. Press **Enter** to send the message.

- **Message Actions**: While a message is selected in the Chat Area, you can perform the following actions:
  - **Delete Message**: Press **d** to delete the selected message.
  - **Edit Message**: Press **e** to edit the selected message.
  - **Reply to Message**: Press **r** to reply to the selected message.

## Visual Indicators

- **Focus Border**: The currently focused component is highlighted with a border, indicating active focus.

- **Selected Message Highlight**: In the Chat Area, the selected message is displayed with a blue background and white text for clear visibility.

