import { TelegramClient } from 'telegram';
import { StringSession } from 'telegram/sessions/index.js';

//@ts-expect-error
import input from 'input';

import os from 'node:os';

import { config } from 'dotenv';

import path from 'node:path';
import fs from 'node:fs';
import { LogLevel } from 'telegram/extensions/Logger.js';

config();

const apiId = process.env.TELEGRAM_API_ID;
const apiHash = process.env.TELEGRAM_API_HASH;

type Config = {
	session: string;
	skipHelp: boolean;
	skipUpdate: boolean;
}

export const tgCliConfig = getConfig();
const session = tgCliConfig?.['session' as keyof typeof tgCliConfig]
const stringSession = new StringSession(session as string ?? '');

export async function getTelegramClient() {
	const client = new TelegramClient(stringSession, Number(apiId), apiHash!, {
		connectionRetries: 5
	});

	client.setLogLevel(LogLevel.NONE);
	if (session) return client;
	await client.start({
		phoneNumber: async () => {
			const phoneNumber = await input.text('Please enter your number: ');
			return phoneNumber;
		},
		password: async () => {
			const password = await input.text('Please enter your password: ');
			return password;
		},
		phoneCode: async () => {
			const phoneCode = await input.text('Please enter the code you received: ');
			return phoneCode;
		},
		onError: (err) => console.error('Authentication error:', err)
	});
	const telegramSession = client.session.save();

	const configData = `session=${telegramSession}`;

	setUserConfigration(configData);
	return client;
}
function getConfigFilePath() {
	const homeDir = os.homedir();

	const configDir = path.join(homeDir, '.tg-cli');
	const configFile = path.join(configDir, 'config.txt');

	return [configFile, configDir] as [string, string];
}

function setUserConfigration(configData: string) {
	const [configFile, configDir] = getConfigFilePath();

	if (!fs.existsSync(configDir)) {
		fs.mkdirSync(configDir, { recursive: true });
	}

	fs.writeFile(configFile, configData, (err) => {
		if (err) {
			console.error('Error writing to config file:', err);
		} else {
			console.log('Configuration file created successfully at', configFile);
		}
	});
}

export function getConfig(): Config | null {
	const [configFile] = getConfigFilePath();
	if (!fs.existsSync(configFile)) {
		return null;
	}
	const configContent = fs.readFileSync(configFile, 'utf-8');
	const configEntries = configContent.split(';');
	const config: Config = {
		session: '',
		skipHelp: false,
		skipUpdate: false
	}  

	configEntries.forEach((entry) => {
		const [key, value] = entry.split('=') as [keyof typeof config, string];
		//@ts-ignore
		config[key] = value as unknown as Config[keyof Config];
	});

	return config as Config;
}


export const setConfig = (key: keyof Config, value: string | boolean) => {
	const config = getConfig();
	const configToWrite = config ? { ...config, ...{ [key]: value } } : { [key]: value }


	//@ts-ignore
	config[key] = value;
	const configData = Object.entries(configToWrite)
		.map(([key, value]) => `${key}=${value}`)
		.join(';');
	setUserConfigration(configData);
}