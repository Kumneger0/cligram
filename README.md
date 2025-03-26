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
- **u**: Switch back to users (Sidebar specific).

## Doing Things

- **Message Actions**: When you've got a message selected in the chat area:
  - **d** to delete it.
  - **e** to edit it.
  - **r** to reply to it.
  - **f** to forward it.

## Configuration Management

customize your cligram experience by managing your own configuration using a JSON file. The configuration file is located at `~/.cligram/user.config.json`.

### Configuration Options

Create or edit the `user.config.json` file with your preferred settings:

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
The configuration file allows you to customize:

#### Chat Settings
- `sendTypingState`: (boolean) Whether to show "typing..." status when composing messages
- `readReceiptMode`: Can be "default", "instant", or "never"
  - `default`: Only send read state when user interacts (replies, etc.)
  - `instant`: Send read state as soon as message is opened
  - `never`: Never send read state, even when replying

#### Privacy Settings
- `lastSeenVisibility`: Controls who can see your last seen status
  - `everyone`: Visible to all users
  - `contacts`: Only visible to contacts
  - `nobody`: Hidden from everyone

#### Notification Settings
- `enabled`: (boolean) Whether to show notifications
- `showMessagePreview`: (boolean) Whether to show message content in notifications

If the configuration file is not present or contains invalid settings, cligram will use default settings. You can modify the configuration file at any time - changes will take effect the next time you start the application.

## Contributing

We welcome contributions to cligram! For detailed guidelines, please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file.

If you encounter any issues or have suggestions, feel free to open an issue on our GitHub repository. This is also a great way to contribute to the project.

Thank you for your interest in improving cligram!

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
