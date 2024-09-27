#!/usr/bin/env node
import blessed from 'blessed';
import contrib from 'blessed-contrib';
import { Api, TelegramClient } from 'telegram';
import { MessagesResponse, MessagesSlice, User } from './lib/types/index.js';
import { getTelegramClient } from './lib/utils/auth.js';
import { LogLevel } from 'telegram/extensions/Logger.js';
let client: Awaited<ReturnType<typeof getTelegramClient>>;

let screen: blessed.Widgets.Screen;
let grid: contrib.grid;
let inputBox: blessed.Widgets.InputElement;

let chatUsers: {

  firstName: string;
  isBot: boolean;
  peerId: bigInt.BigInteger;
  accessHash: bigInt.BigInteger;
}[] = [];

interface FormattedMessage {
  sender: string;
  content: string;
  isFromMe: boolean;
}

const eventClassNames = ['UpdateUserStatus', 'UpdateShortMessage'] as const;

let chatBox: blessed.Widgets.ListElement;
let currentLine = 0;

let selectedName: string;
try {
  getTelegramClient().then(async (result) => {
    listenForUserMessages(result);
    client = result;
    client.setLogLevel(LogLevel.NONE);
    screen = blessed.screen({
      smartCSR: true,
      title: 'Terminal Telegram client'
    });

    grid = new contrib.grid({ rows: 12, cols: 12, screen: screen });
    initializeApp();
  });
} catch (err) {
  console.error(err);
  process.exit(1);
}

function initializeApp() {
  initializeMainInterface();
}

let sidebar: blessed.Widgets.ListElement;

async function initializeMainInterface() {
  const chats = await getUserChats();
  sidebar = grid.set(0, 0, 12, 3, blessed.list, {
    label: 'Names',
    items: chats.map((user) => user?.firstName ?? null).filter(Boolean),
    keys: true,
    mouse: true,
    style: {
      selected: {
        bg: 'blue',
        fg: 'white'
      }
    }
  });

  chatBox = grid.set(0, 3, 10, 9, blessed.list, {
    label: 'Chat',
    scrollable: true,
    alwaysScroll: true,
    scrollbar: {
      ch: ' ',
      inverse: true
    },
    keys: true,
    vi: true,
    mouse: true,
    border: {
      type: 'line'
    },
    style: {
      fg: 'white',
      border: {
        fg: 'cyan'
      },
      selected: {
        bg: 'blue',
        fg: 'white'
      }
    },
    tags: true
  }) as blessed.Widgets.ListElement;


  inputBox = grid.set(
    10,
    3,
    2,
    9,
    blessed.textbox,
    {
      inputOnFocus: true,
      border: {
        type: "line",
      },
      style: {
        fg: "white",
        border: {
          fg: "cyan",
        },
      }
    }
  );

  sidebar.on('select', async (item: blessed.Widgets.ListElement) => {
    selectedName = item.getText();
    chatBox.setLabel(selectedName);
    const user = chatUsers.find(({ firstName }) => firstName === selectedName);
    if (user) {
      const conversation = await getConversationHistory(user);
      updateChatBox(conversation);
      addKeyBindingToFocusOnInputBox();
    }
    chatBox.focus();
    screen.render();
  });

  inputBox.key(['enter'], async () => {
    const value = inputBox.content;

    if (!client.connected) await client.connect();

    const userToSend = chatUsers.find(({ firstName }) => firstName == selectedName);
    if (!value || !userToSend) {
      chatBox.focus();
      return;
    }

    client.sendMessage(
      new Api.InputPeerUser({
        userId: userToSend?.peerId,
        accessHash: userToSend?.accessHash
      }),
      {
        message: value
      }
    );

    screen.render();
  });

  // Add this key handler for the chatBox
  chatBox.key(['up', 'down'], function (ch, key) {
    const direction = key.name === 'up' ? -1 : 1;
    currentLine = Math.max(0, Math.min(currentLine + direction, chatBox.getLines().length - 1));
    updateChatBox();
    screen.render();
  });

  async function getConversationHistory(
    { accessHash, firstName, peerId: userId }: (typeof chatUsers)[number],
    limit: number = 100
  ): Promise<FormattedMessage[]> {
    const client = await getTelegramClient();

    if (!client.connected) await client.connect();

    const result = (await client.invoke(
      new Api.messages.GetHistory({
        peer: new Api.InputPeerUser({
          userId: userId,
          accessHash
        }),
        limit
      })
    )) as unknown as MessagesSlice;

    return result.messages.reverse().map(
      (message): FormattedMessage => ({
        sender: message.out ? 'you' : firstName,
        content: message.message,
        isFromMe: message.out
      })
    );
  }

  function updateChatBox(newContent?: FormattedMessage[]) {
    if (newContent) {
      const formattedMessages = newContent.flatMap((msg, index) => formatMessage(msg, index));
      (chatBox as blessed.Widgets.ListElement).setItems(formattedMessages);
      (chatBox as blessed.Widgets.ListElement).scrollTo(formattedMessages.length - 1);
    }
    screen.render();
  }

  function formatMessage(message: FormattedMessage, index: number): string[] {
    const { sender, content, isFromMe } = message;
    const padding = isFromMe ? '' : '  ';
    const header = `${padding}${sender}:`;
    const wrappedContent = wrapText(content, MAX_WIDTH - padding.length);

    return [
      header,
      ...wrappedContent.map(line => `${padding}${line}`),
      '' // Empty line for spacing between messages
    ];
  }

  screen.key(['escape', 'q', 'C-c'], function () {
    return process.exit(0);
  });

  inputBox.key(['C-h'], () => {
    sidebar.focus();
  });

  let focusOnSidebar = true;

  screen.key(['tab'], () => {
    if (focusOnSidebar) {
      chatBox.focus();
    } else {
      sidebar.focus();
    }
    focusOnSidebar = !focusOnSidebar;
    screen.render();
  });

  sidebar.focus();
  screen.render();
}

async function getUserChats() {
  if (!client.connected) await client.connect();

  const result = (await client.invoke(
    new Api.messages.GetDialogs({
      offsetDate: 0,
      offsetId: 0,
      offsetPeer: new Api.InputPeerEmpty(),
      limit: 3000
    })
  )) as unknown as MessagesResponse;

  const clipboardy = (await import('clipboardy')).default;

  const userChats = result.dialogs.filter((dialog) => dialog.peer.className === 'PeerUser');
  // const userChats = result.dialogs.filter((dialog) => dialog.className === "User");

  clipboardy.writeSync(JSON.stringify(userChats, null, 2));

  const users = await Promise.all(
    userChats.map(async ({ peer }) => {
      try {
        const user = (await getUserInfo(peer.userId)) as unknown as User;
        if (!user) return null;
        return {
          firstName: user.firstName,
          isBot: user.bot,
          peerId: peer.userId,
          accessHash: user.accessHash as unknown as bigInt.BigInteger
        };
      } catch (err) {
        console.error(err);
        return null;
      }
    })
  );

  chatUsers = users.filter((user): user is NonNullable<typeof user> => user !== null);

  return chatUsers.filter(({ isBot }) => !isBot);
}

async function getUserInfo(userId: bigInt.BigInteger) {
  try {
    if (!client.connected) await client.connect();
    const user = await client.getEntity(await client.getInputEntity(userId));
    return user;
  } catch (err) {
    console.error(err);
  }
}

const listenForUserMessages = async (client: TelegramClient) => {
  if (!client.connected) await client.connect();
  console.log('Listening for messages');

  client.addEventHandler(async (event) => {
    const userId = event.userId;
    if (userId) {
      const isNewMessage =
        (event.className as (typeof eventClassNames)[number]) === 'UpdateShortMessage';

      if (isNewMessage) {
        const user = (await getUserInfo(userId)) as unknown as User;

        if (user.firstName !== selectedName) {
          {
            const users = await getUserChats();
            const userChats = users
              .filter(({ isBot }) => !isBot)
              .map(({ firstName }) => firstName)
              .map((name) => (name === selectedName ? name + ' *' : name));

            rerenderSidebar(userChats);
          }
        }
      }
    }
  });
};

function rerenderSidebar(items: string[]) {
  screen.remove(sidebar);

  sidebar = grid.set(0, 0, 12, 3, blessed.list, {
    label: 'Names',
    items,
    keys: true,
    mouse: true,
    style: {
      selected: {
        bg: 'blue',
        fg: 'white'
      }
    }
  });
  screen.render();
}

const MAX_WIDTH = 50; // Approximate character width equivalent to 500px

function wrapText(text: string, maxWidth: number): string[] {
  const words = text.split(' ');
  const lines: string[] = [];
  let currentLine = '';

  words.forEach(word => {
    if (currentLine.length + word.length + 1 > maxWidth) {
      lines.push(currentLine);
      currentLine = word;
    } else {
      currentLine += (currentLine ? ' ' : '') + word;
    }
  });

  if (currentLine) {
    lines.push(currentLine);
  }

  return lines;
}


function addKeyBindingToFocusOnInputBox() {
  if (chatBox && inputBox) {
    chatBox.key(['i'], () => {
      if (selectedName) {
        inputBox.focus();
        screen.render();
      }
    });
  }
}

