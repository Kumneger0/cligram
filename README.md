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

- **Sidebar Navigation**: When the Sidebar is focused, use the **Up Arrow** and **Down Arrow** keys to move through the chat list.

- **Chat Area Navigation**: When the Chat Area is focused, navigate through the conversation history using the **Up Arrow** and **Down Arrow** keys. The currently selected message is highlighted with a blue background and white text.

## User Interaction

- **Input Field**: Focus on the Input Field to type your message. Press **Enter** to send the message.

- **Message Actions**: While a message is selected in the Chat Area, you can perform the following actions:
  - **Delete Message**: Press **D** to delete the selected message.
  - **Edit Message**: Press **E** to edit the selected message.
  - **Reply to Message**: Press **R** to reply to the selected message.

## Visual Indicators

- **Focus Border**: The currently focused component is highlighted with a border, indicating active focus.

- **Selected Message Highlight**: In the Chat Area, the selected message is displayed with a blue background and white text for clear visibility.

This design ensures an intuitive user experience, allowing efficient navigation and interaction within the CLI application.
