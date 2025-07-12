import { TelegramClient } from 'telegram';
import { RPCError } from 'telegram/errors/index.js';
import { StringSession } from 'telegram/sessions/index.js';

import { config } from 'dotenv';
import input from 'input';
import os from 'node:os';

import { red } from 'kolorist';
import fs, { writeFileSync } from 'node:fs';
import path from 'node:path';
import { LogLevel } from 'telegram/extensions/Logger.js';

config();

const apiId = process.env.TELEGRAM_API_ID;
const apiHash = process.env.TELEGRAM_API_HASH;

type Config = {
	session: string;
	skipHelp: boolean;
	skipUpdate: boolean;
};

export const tgCliConfig = getConfig();
const session = tgCliConfig?.['session' as keyof typeof tgCliConfig];
const stringSession = new StringSession((session ?? '') as string);

let telegramClient: TelegramClient;

export async function authenticateUser({isCalledFromLogin}:{ isCalledFromLogin: boolean }): Promise<TelegramClient | null> {
	const client = new TelegramClient(stringSession, Number(apiId), apiHash!, {
		connectionRetries: 5
	});
	client.setLogLevel(LogLevel.NONE);
	try {
		if (session) {
			return client;
		}
		if (!isCalledFromLogin && !session) {
			throw Error("Are u logged in ?")
		}
		await client.start({
			phoneNumber: async () => {
				const phoneNumber = await (input as { text: (prompt: string) => Promise<string> }).text(
					'Please enter your number: '
				);
				return phoneNumber;
			},
			password: async () => {
				const password = await (
					input as { password: (prompt: string) => Promise<string> }
				).password('Please enter your password: ');
				return password;
			},
			phoneCode: async () => {
				const phoneCode = await (input as { text: (prompt: string) => Promise<string> }).text(
					'Please enter the code you received: '
				);
				return phoneCode;
			},
			onError: (err) => {
				if (err.message.includes('PHONE_CODE_INVALID')) {
					console.error('Error: The code you entered is incorrect. Please try again.');
				} else if (err.message.includes('PASSWORD_HASH_INVALID')) {
					console.error('Error: The password you entered is incorrect. Please try again.');
				} else {
					console.error('Authentication error:', err.message);
				}
			}
		});
		const telegramSession = await client.session.save();

		//@ts-ignore
		if (!telegramSession) {
			throw new Error('Failed to login to telegram');
		}
		const configData = `session=${telegramSession}`;
		setUserConfigration(configData);
		return client;
	} catch (err) {
		if (err instanceof RPCError) {
			const message = err.message;
			const errMessage = `${red('✖')} ${message}`;
			process.stdout.write(errMessage);
			return null;
		}
		if (err && typeof err === 'object' && 'message' in err && typeof err.message === 'string') {
			const errMessage = `${red('✖')} ${err.message}`;
			process.stdout.write(errMessage);
		}
	}
	return null;
}


export const getTelegramClient = async (isCalledFromLogin:boolean = false) => {
	if (!telegramClient) {
		const result = await authenticateUser({isCalledFromLogin});
		if (!result) {
			throw new Error('Failed to authetiate user');
		}
		telegramClient = result;
	}
	return telegramClient;
};

function getConfigFilePath() {
	const homeDir = os.homedir();
	const configDir = path.join(homeDir, '.cligram');
	const configFile = path.join(configDir, 'config.txt');
	return [configFile, configDir];
}

export function removeConfig() {
	try {
		const homeDir = os.homedir();
		const configDir = path.join(homeDir, '.cligram');
		const configDirExists = fs.existsSync(configDir);
		if (configDirExists) {
			fs.rmSync(configDir, { recursive: true });
			return true;
		}
	} catch (err) {
		console.error(`${red('✖')} ${(err as Error).message}`);
	}
}

function setUserConfigration(configData: string) {
	const [configFile, configDir] = getConfigFilePath();

	if (!configFile || !configDir) {
		console.error('Error: Config file or directory not found');
		return;
	}

	if (!fs.existsSync(configDir)) {
		fs.mkdirSync(configDir, { recursive: true });
	}

	fs.writeFile(configFile, configData, (err) => {
		if (err) {
			if (typeof err === 'object' && 'message' in err && typeof err.message === 'string') {
				console.log(`${red('✖')} ${err.message}`);
				return;
			}
			console.error('Error writing to config file:', err);
		}
	});
}

export function getConfig(): Config | null {
	const [configFile] = getConfigFilePath();

	if (!configFile) {
		console.error('Error: Config file not found');
		return null;
	}

	if (!fs.existsSync(configFile)) {
		return null;
	}
	const configContent = fs.readFileSync(configFile, 'utf-8');
	const configEntries = configContent.split(';');
	const config: Config = {
		session: '',
		skipHelp: false,
		skipUpdate: false
	};

	configEntries.forEach((entry) => {
		const [key, value] = entry.split('=') as [keyof typeof config, string];
		//@ts-ignore
		config[key] = value as unknown as Config[keyof Config];
	});

	return config as Config;
}

export const setConfig = (key: keyof Config, value: string | boolean) => {
	const config = getConfig();
	const configToWrite = config ? { ...config, ...{ [key]: value } } : { [key]: value };

	//@ts-ignore
	config[key] = value;
	const configData = Object.entries(configToWrite)
		.map(([key, value]) => {
			return `${key}=${value}`;
		})
		.join(';');
	setUserConfigration(configData);
};
