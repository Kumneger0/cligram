#!/usr/bin / env node
import { getTelegramClient } from './lib/utils/auth.js';
import { LogLevel } from 'telegram/extensions/Logger.js';
import { initializeUI } from './ui/initializeUI.js';
import { listenForUserMessages } from './telegram/messages.js';
try {
  getTelegramClient().then(async (client) => {
    client.setLogLevel(LogLevel.NONE);
    listenForUserMessages(client);
    initializeUI(client);
  });
} catch (err) {
  console.error(err);
  process.exit(1);
}
