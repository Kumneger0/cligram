#!/usr/bin / env node
import { LogLevel } from 'telegram/extensions/Logger';
import { getTelegramClient } from './lib/utils/auth';
// import { initializeUI } from './ui/initializeUI';
import { initializeUI } from './main';
try {
	getTelegramClient().then(async (client) => {
		client.setLogLevel(LogLevel.NONE);
		initializeUI(client);
	});
} catch (err) {
	console.error(err);
	process.exit(1);
}
