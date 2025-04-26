#!/usr/bin/env bun
import { cli } from 'cleye';
import { red } from 'kolorist';
import { TelegramClient } from 'telegram';
import { RPCError } from 'telegram/errors/index.js';
import { LogLevel } from 'telegram/extensions/Logger.js';
import { description, version } from '../package.json';
import { login, logout } from './commands';
import { getTelegramClient, removeConfig } from './lib/utils/auth';
import { initializeUI } from './main';
import { setUserPrivacy } from './telegram/client';

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
		ignoreArgv: (type) => {
			return type === 'unknown-flag' || type === 'argument';
		}
	},
	async (_argv) => {
		try {
			const client = await getTelegramClient();
			if (client) {
				await client.connect();
				const me = await client.getMe();
				if (me.phone) {
					client.setLogLevel(LogLevel.NONE);
					await setUserPrivacy(client);
					const root = await initializeUI(client);
					for (const signal of ['SIGINT', 'SIGTERM']) {
						process.on(signal, () => {
							disconnect(client);
							root.cleanup();
							console.log('Cleaning up');
						});
					}
					return;
				}
			}
			console.log(`${red('✖')} Are you logged in ?`);
			console.log('login with cligram login');
			process.exit(0);
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
