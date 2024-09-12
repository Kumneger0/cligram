#!/usr/bin/env node

import blessed from "blessed";
import contrib from "blessed-contrib";
import { getTelegramClient } from "./lib/utils/auth.js";
import { Api } from "telegram";
import { Dialog, MessagesResponse } from "./lib/types/index.js";

// Create a button to initiate sign-in.

let client: Awaited<ReturnType<typeof getTelegramClient>>;

const demoMessages = ` \nAlice: Hi there!
    \nBob: Hello Alice, how are you?
    \nAlice: I'm good, thanks! How about you?
    \nBob: I'm doing well. Just working on some projects.`;

let screen: blessed.Widgets.Screen;
let grid: contrib.grid;

try {
  getTelegramClient().then((result) => {
    client = result;
    screen = blessed.screen({
      smartCSR: true,
      title: "CLI App with Sidebar",
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

async function initializeMainInterface() {
  const chats = await getUserChats();
  const sidebar: blessed.Widgets.BlessedElement = grid.set(
    0,
    0,
    12,
    3,
    blessed.list,
    {
      label: "Names",
      items: chats.map(({ firstName }) => firstName),
      keys: true,
      mouse: true,
      style: {
        selected: {
          bg: "blue",
          fg: "white",
        },
      },
    }
  );

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

  sidebar.on("select", (item: blessed.Widgets.ListElement) => {
    const selectedName = item.getText();
    chatBox.setLabel(selectedName);
    inputBox.focus();
    chatBox.setContent(demoMessages);
    screen.render();
  });

  screen.key(["escape", "q", "C-c"], function () {
    return process.exit(0);
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
      offsetPeer: new Api.InputPeerUser({ userId: 0 }),
      limit: 10,
    })
  )) as unknown as MessagesResponse;

  const userChats = result.dialogs.filter(
    (dialog) => dialog.peer.className === "PeerUser"
  );

  const users = userChats.map(({ peer }) => {
    const firstName = peer.userId;
    return {
      firstName,
    };
  });
  return users;
}

async function getUserInfo(userId: number) {
  const user = await client.getEntity({
    userId: userId,
  });
}
