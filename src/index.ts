#!/usr/bin/env node
import blessed from "blessed";
import contrib from "blessed-contrib";
import { Api, TelegramClient } from "telegram";
import { MessagesResponse, MessagesSlice, User } from "./lib/types/index.js";
import { getTelegramClient } from "./lib/utils/auth.js";
import { LogLevel } from "telegram/extensions/Logger.js";
let client: Awaited<ReturnType<typeof getTelegramClient>>;

let screen: blessed.Widgets.Screen;
let grid: contrib.grid;

let chatUsers: {
  firstName: string;
  isBot: boolean;
  peerId: bigInt.BigInteger;
  accessHash: bigInt.BigInteger;
}[] = [];

const eventClassNames = ["UpdateUserStatus", "UpdateShortMessage"] as const;

let selectedName: string;
try {
  getTelegramClient().then(async (result) => {
    listenForUserMessages(result);
    client = result;
    client.setLogLevel(LogLevel.NONE);
    screen = blessed.screen({
      smartCSR: true,
      title: "Terminal Telegram client",
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

let sidebar: blessed.Widgets.BlessedElement;

async function initializeMainInterface() {
  const chats = await getUserChats();
  sidebar = grid.set(0, 0, 12, 3, blessed.list, {
    label: "Names",
    items: chats.map((user) => user?.firstName ?? null).filter(Boolean),
    keys: true,
    mouse: true,
    style: {
      selected: {
        bg: "blue",
        fg: "white",
      },
    },
  });

  const chatBox: blessed.Widgets.BlessedElement = grid.set(
    0,
    3,
    12,
    9,
    blessed.box,
    {
      label: "Chat",
      content: `
    Welcome to the Terminal Telegram Client!
   
  `,
      scrollable: true,
      alwaysScroll: true,
      border: {
        type: "line",
      },
      style: {
        fg: "white",
        border: {
          fg: "cyan",
        },
      },
    }
  );

  const inputBox: blessed.Widgets.InputElement = grid.set(
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
      },
    }
  );

  sidebar.on("select", async (item: blessed.Widgets.ListElement) => {
    selectedName = item.getText();
    chatBox.setLabel(selectedName);
    const user = chatUsers.find(({ firstName }) => firstName === selectedName);
    if (user) {
      const conversation = await getConversationHistory(user);
      chatBox.setContent("");
      chatBox.setContent(conversation);
    }
    inputBox.focus();
    screen.render();
  });

  inputBox.key(["enter"], async () => {
    const value = inputBox.content;

    if (!client.connected) await client.connect();

    const userToSend = chatUsers.find(
      ({ firstName }) => firstName == selectedName
    );
    if (!value || !userToSend) {
      chatBox.focus();
      return;
    }

    client.sendMessage(
      new Api.InputPeerUser({
        userId: userToSend?.peerId,
        accessHash: userToSend?.accessHash,
      }),
      {
        message: value,
      }
    );

    screen.render();
  });

  async function getConversationHistory(
    { accessHash, firstName, peerId: userId }: (typeof chatUsers)[number],
    limit: number = 100
  ) {
    const client = await getTelegramClient();

    if (!client.connected) await client.connect();

    const result = (await client.invoke(
      new Api.messages.GetHistory({
        peer: new Api.InputPeerUser({
          userId: userId,
          accessHash,
        }),
        limit,
      })
    )) as unknown as MessagesSlice;

    const formattedMessages = result.messages
      .reverse()
      .map((message) => {
        const senderName = !message.out ? firstName : "you";
        return `${senderName}: ${message.message}`;
      })
      .join("\n");

    return formattedMessages;
  }

  screen.key(["escape", "q", "C-c"], function () {
    return process.exit(0);
  });

  inputBox.key(["C-h"], () => {
    sidebar.focus();
  });

  let focusOnSidebar = true;

  screen.key(["tab"], () => {
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
      limit: 3000,
    })
  )) as unknown as MessagesResponse;

  const clipboardy = (await import("clipboardy")).default;

  const userChats = result.dialogs.filter(
    (dialog) => dialog.peer.className === "PeerUser"
  );
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
          accessHash: user.accessHash as unknown as bigInt.BigInteger,
        };
      } catch (err) {
        console.error(err);
        return null;
      }
    })
  );

  chatUsers = users.filter(
    (user): user is NonNullable<typeof user> => user !== null
  );

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
  console.log("Listening for messages");

  client.addEventHandler(async (event) => {
    const userId = event.userId;
    if (userId) {
      const isNewMessage =
        (event.className as (typeof eventClassNames)[number]) ===
        "UpdateShortMessage";

      if (isNewMessage) {
        const user = (await getUserInfo(userId)) as unknown as User;

        if (user.firstName !== selectedName) {
          {
            const users = await getUserChats();
            const userChats = users
              .filter(({ isBot }) => !isBot)
              .map(({ firstName }) => firstName)
              .map((name) => (name === selectedName ? name + " *" : name));

            rerenderSidebar(userChats);
          }

          // if(name === selectedName) {
          //   const conversation = await getConversationHistory(user);
          //   chatBox.setContent(conversation);
          // }

          // console.log("event", event);
          // const message = event.message;
          // console.log(
          //   "user firstName",
          //   user?.firstName ?? "user doesn't have a name"
          // );
          // console.log("message", message);
        }
      }
    }
  });
};

function rerenderSidebar(items: string[]) {
  screen.remove(sidebar);

  sidebar = grid.set(0, 0, 12, 3, blessed.list, {
    label: "Names",
    items,
    keys: true,
    mouse: true,
    style: {
      selected: {
        bg: "blue",
        fg: "white",
      },
    },
  });
  screen.render();
}
