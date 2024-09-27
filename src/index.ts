#!/usr/bin / env node
import { getTelegramClient } from './lib/utils/auth.js';
import { LogLevel } from 'telegram/extensions/Logger.js';
import { initializeUI } from './ui/initializeUI.js';
import { listenForUserMessages } from './telegram/messages.js';

let client: Awaited<ReturnType<typeof getTelegramClient>>;

try {
  getTelegramClient().then(async (result) => {
    client = result;
    client.setLogLevel(LogLevel.NONE);
    listenForUserMessages(client);
    initializeUI(client);
  });
} catch (err) {
  console.error(err);
  process.exit(1);
}