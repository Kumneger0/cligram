# CliGram

This is a Telegram CLI client made with TypeScript and Node.js.

**Note:** This project is currently in development and is not fully stable. expect potential bugs and incomplete features.

Right now, you can only chat with personal chats and channels. Group and bot support is coming soon!

## How to Use It

When you start it up, you'll see three main parts:

1.  **Sidebar**: Lists all your personal chats (takes up about 30% of your screen).
2.  **Chat Area**: Where the chat messages show up (takes up the other 70%).
3.  **Help Page**: The first thing you'll see, it tells you how to get around.

## Initial Setup

First, get your `api_id` and `api_hash` from [Telegram](https://my.telegram.org/apps).

### Set environment Variables

If you're on a Unix-based system like Linux or macOS, set your `api_id` and `api_hash` in your `.zshrc` or `.bashrc` file:

```bash
export TELEGRAM_API_ID=your_api_id_from_telegram
export TELEGRAM_API_HASH=your_api_hash_from_telegram
```

### Set environment Variables (Windows)

go figure out by yourself

### Installation with npm

You'll need to have `bun` installed to use this package:

```bash
npm install -g cligram
```

## Using Docker

Here's the Docker command to run the app:

```bash
docker run --rm -it -v tele_cli_data:/root/.cligram -e TELEGRAM_API_ID=$TELEGRAM_API_ID -e TELEGRAM_API_HASH=$TELEGRAM_API_HASH kumneger/cligram:latest
```

### Why is the Docker Command Long?

By default, cligram stores the user's session information in a hidden folder called `.cligram` in the user's home directory. Docker containers have their own file system, which is isolated from the host machine. To prevent the need for re-authenticating the user every time, we create a Docker volume and bind it to the container. This allows us to persist the session information across container restarts.

Additionally, we pass the Telegram API ID and API Hash as environment variables (`TELEGRAM_API_ID` and `TELEGRAM_API_HASH`). These are required for cligram to authenticate and interact with the Telegram API.

If the command is too long, you can create an alias to make it easier to use.

## Before You Start

- **Login**: Use `cligram login` in your terminal and follow the prompts.
- **Logout**: Use `cligram logout` when you're done.

## First Time?

When you start, you'll land on the Help Page. You can:

- Hit **c** to jump into the main chat interface.
- Hit **x** to skip the help page next time.

## The Look

- Sidebar takes 30% of the width.
- Chat area fills the rest.
- Uses the full height of your terminal.
- Nice rounded borders separate everything.

## Getting Around

- **Tab**: Switch between the sidebar and the chat area (active section has a green border).
- **↑** or **k**: Move up (works in both chat list and messages).
- **↓** or **j**: Move down (works in both chat list and messages).
- **ctrl + k** : To Open up search menu
- **c**: Switch to Channels (Sidebar specific).
- **g**: Switch to Groups(Sidebar specific))
- **u**: Switch back to users (Sidebar specific).

## Doing Things

- **Message Actions**: When you've got a message selected in the chat area:
  - **d** to delete it.
  - **e** to edit it.
  - **r** to reply to it.
  - **f** to forward it.
  - **u** to open direct message with the user (in group chats, this lets you quickly start a private conversation with any message sender)


## Working with the Message Input

- **ctrl + x**: Toggle focus on the message input box
- **ctrl + a**: Open file picker to attach a file (requires zenity to be installed)
- When a file is selected, you can add an optional caption or just press Enter to send the file as is
- File upload progress is displayed while sending

## Configuration Management

customize your cligram experience by managing your own configuration using a JSON file. The configuration file is located at `~/.cligram/user.config.json`.

### Configuration Options

All configuration options are optional. Here are all available options with their possible values and defaults:

```json
{
  "chat": {
    // Whether to show "typing..." status when composing messages
    // Possible values: true, false
    // Default: true
    "sendTypingState": true,

    // Controls when to send read receipts
    // Possible values: "default", "instant", "never"
    // Default: "default"
    "readReceiptMode": "default"
  },
  "privacy": {
    // Who can see your last seen status
    // Possible values: "everyone", "contacts", "nobody"
    // Default: undefined (uses Telegram's default setting)
    "lastSeenVisibility": "everyone"
  },
  "notifications": {
    // Whether to show notifications
    // Possible values: true, false
    // Default: true
    "enabled": true,

    // Whether to show message content in notifications
    // Possible values: true, false
    // Default: true
    "showMessagePreview": true
  }
}
```

Here's what each setting does:

#### Chat Settings

- `sendTypingState`: Controls whether others see "typing..." when you're composing a message
  - `true`: Show typing status (default)
  - `false`: Never show typing status
- `readReceiptMode`: Controls when messages are marked as read
  - `"default"`: Only marks messages as read when you actively interact with them (like replying). the sender won't see the "read" checkmarks until you take action on their messages.
  - `"instant"`: Marks messages as read immediately when you view them in the chat area. The sender will see "read" checkmarks as soon as you look at their messages.
  - `"never"`: Messages are never automatically marked as read, even when you interact with them. The sender will always see their messages with unread status.

#### Privacy Settings

- `lastSeenVisibility`: Controls who can see when you were last online
  - `"everyone"`: Anyone can see your last seen time
  - `"contacts"`: Only your contacts can see your last seen time
  - `"nobody"`: No one can see your last seen time
  - If not set: Uses your existing Telegram privacy settings

#### Notification Settings

- `enabled`: Master switch for notifications
  - `true`: Show notifications (default)
  - `false`: Disable all notifications
- `showMessagePreview`: Controls notification content
  - `true`: Show message content in notifications (default)
  - `false`: Only show sender name, hide message content

All settings are optional - if you omit any setting, cligram will use the default value. You can modify the configuration file at any time - changes will take effect the next time you start the application.

## Contributing

We welcome contributions to cligram! For detailed guidelines, please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file.

If you encounter any issues or have suggestions, feel free to open an issue on our GitHub repository. This is also a great way to contribute to the project.

Thank you for your interest in improving cligram!

## 🤝 Code of Conduct

We are committed to providing a welcoming and inclusive experience for everyone. We expect all participants in our community to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it to understand what behaviors will and will not be tolerated.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
