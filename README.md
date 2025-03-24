# Telegram CLI

This is a Telegram CLI client made with TypeScript and Node.js.

**Heads Up:** Right now, you can only chat with personal chats and channels. Group and bot support is coming soon!

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
###  Set environment Variables (Windows)
 go figure out by yourself

### Installation with npm

You'll need to have `bun` installed to use this package:

```bash
npm install -g cligram
```

## Using Docker

Here's the Docker command to run the app:

```bash
docker run --rm -it -v tele_cli_data:/root/.tg-cli -e TELEGRAM_API_ID=$TELEGRAM_API_ID -e TELEGRAM_API_HASH=$TELEGRAM_API_HASH kumneger/cligram:latest
```

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
- **ctrl + k**  : To Open up search menu
- **c**: Switch to Channels (Sidebar specific).
- **u**: Switch back to users (Sidebar specific).

## Doing Things

- **Message Actions**: When you've got a message selected in the chat area:
  - **d** to delete it.
  - **e** to edit it.
  - **r** to reply to it.
  - **f** to forward it.

## Contributing

We welcome contributions to cligram! For detailed guidelines, please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file.

If you encounter any issues or have suggestions, feel free to open an issue on our GitHub repository. This is also a great way to contribute to the project.

Thank you for your interest in improving cligram!


## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

