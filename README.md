# Telegram CLI

Telegram CLI client built with TypeScript and Node.js.

**Important Note:** Right now, you can only chat with personal chats. Groups, channels, and bots aren't supported yet, but I'm planning to add them in the future!

## How to Use It

When you fire it up, you get a cool interactive setup with three main parts:

1.  **Sidebar**: That's where you see all your personal chats listed (takes up about 30% of your screen).
2.  **Chat Area**: This is where the actual chat messages show up (takes up the other 70%).
3.  **Help Page**: The first thing you'll see, it tells you how to get around.

## Initial Setup & Login/Logout

Before you can use the app, you'll need to login.

- **Login**: Use the command `tele-cli login` in your terminal. Follow the prompts to authenticate.
- **Logout**: When you're done, use the command `tele-cli logout` to log out.

## First Time?

When you start, you'll land on the Help Page. You've got two choices:

- Hit **c** to jump right into the main chat interface.
- Hit **x** to go straight to the interface and skip the help page next time.

## The Look

It's designed to fit your terminal size, so:

- The sidebar gets 30% of the width.
- The chat area fills the rest.
- It uses the full height of your terminal.
- You'll see nice rounded borders separating everything.

## Getting Around

- **Switching Sides**: Press **Tab** to jump between the sidebar and the chat area.
- The section you're on has a green border, so you know where you are.

### Sidebar Stuff

- Use **↑** or **k** to move up the chat list.
- Use **↓** or **j** to move down.

### Chat Area Stuff

- **Moving Through Messages:**
  - **↑** or **k** for older messages.
  - **↓** or **j** for newer messages.

## Doing Things

- **Message Actions**: When you've got a message selected in the chat area:
  - **d** to delete it.
  - **e** to edit it.
  - **r** to reply to it.

## What You'll See

- **Green Borders**: The different parts have green borders.
- **Resizing**: It changes size automatically when you change your terminal window.
