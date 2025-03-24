#!/usr/bin/env bun
import { LogLevel } from 'telegram/extensions/Logger.js';
import { getTelegramClient, removeConfig } from './lib/utils/auth';
import { initializeUI } from './main';
import { cli } from 'cleye';
import { version, description } from '../package.json';
import { login, logout } from './commands';
import { RPCError } from 'telegram/errors/index.js';
import { red } from 'kolorist';
import { TelegramClient } from 'telegram';

const rawArgv = process.argv.slice(2);
const disconnect = async (client: TelegramClient) => {
	await client.disconnect();
	process.exit(0);
};

cli(
	{
		name: 'cligram',
		version,
		commands: [login, logout],
		help: {
			description
		},
		ignoreArgv: (type) => { return type === 'unknown-flag' || type === 'argument' }
	},
	async (_argv) => {
		try {
			const client = await getTelegramClient();
			if (client) {
				await client.connect();
				const me = await client.getMe();
				if (me.phone) {
					for (const signal of ['SIGINT', 'SIGTERM']) {
						process.on(signal, () => {
							console.log('Cleaning up');
							disconnect(client);
						});
					}
					client.setLogLevel(LogLevel.NONE);
					initializeUI(client);
					return;
				}
			}
			console.log(`${red('✖')} Are you logged in ?`);
			console.log('login with cligram login');
		} catch (err) {
			if (err instanceof RPCError) {
				if (err.errorMessage === 'AUTH_KEY_UNREGISTERED') {
					console.log(`${red('✖')} ${err.message}`);
					console.error(
						`${red('✖')} We Are unable to access your chat did u terminated the session ?`
					);
					removeConfig();
					process.exit(0);
				}
				return;
			}
			if (err && typeof err === 'object' && 'message' in err && typeof err.message === 'string') {
				console.log(`${red('✖')} ${err.message}`);
			}
		}
	},
	rawArgv
);
