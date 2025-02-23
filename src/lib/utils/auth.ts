import { TelegramClient } from 'telegram';
import { StringSession } from 'telegram/sessions/index.js';
import { RPCError } from 'telegram/errors'

//@ts-expect-error
import input from 'input';
import os from 'node:os';
import { config } from 'dotenv';

import path from 'node:path';
import fs from 'node:fs';
import { LogLevel } from 'telegram/extensions/Logger.js';
import { red } from 'kolorist';

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

export async function getTelegramClient(isCalledFromLogin = false) {
  const client = new TelegramClient(stringSession, Number(apiId), apiHash!, {
    connectionRetries: 5
  });
  client.setLogLevel(LogLevel.NONE);
  try {
    if (session) return client;
    if (!isCalledFromLogin && !session) return null
    await client.start({
      phoneNumber: async () => {
        const phoneNumber = await input.text('Please enter your number: ');
        return phoneNumber;
      },
      password: async () => {
        const password = await input.password('Please enter your password: ');
        return password;
      },
      phoneCode: async () => {
        const phoneCode = await input.text('Please enter the code you received: ');
        return phoneCode;
      },
      onError: (err) => {
        if (err.message.includes('PHONE_CODE_INVALID')) {
          console.error('Error: The code you entered is incorrect. Please try again.');
        } else if (err.message.includes('PASSWORD_HASH_INVALID')) {
          console.error('Error: The password you entered is incorrect. Please try again.');
        } else {
          console.error('Authentication error:', err);
        }
      }
    });
    const telegramSession = client.session.save();
    const configData = `session=${telegramSession}`;
    setUserConfigration(configData);
    return client;
  } catch (err) {
    if (err instanceof RPCError) {
      const message = err.message
      console.log(`${red('✖')} ${message}`)
      return
    }
    if (err && typeof err == 'object' && 'message' in err && typeof err.message == 'string') {
      console.log(`${red('✖')} ${err.message}`)
    }
  }

}
function getConfigFilePath() {
  const homeDir = os.homedir();
  const configDir = path.join(homeDir, '.tg-cli');
  const configFile = path.join(configDir, 'config.txt');
  return [configFile, configDir] as [string, string];
}

export function removeConfig() {
  try {
    const homeDir = os.homedir();
    const configDir = path.join(homeDir, '.tg-cli');
    const configFile = path.join(configDir, 'config.txt');
    const configFileExists = fs.existsSync(configFile)
    if (configFileExists) {
      fs.rmSync(configFile)
      return true
    }
  } catch (err) {
    console.error(`${red('✖')} ${(err as Error).message}`);
  }
}

function setUserConfigration(configData: string) {
  const [configFile, configDir] = getConfigFilePath();

  if (!fs.existsSync(configDir)) {
    fs.mkdirSync(configDir, { recursive: true });
  }

  fs.writeFile(configFile, configData, (err) => {
    if (err) {
      if (err && typeof err == 'object' && 'message' in err && typeof err.message == 'string') {
        console.log(`${red('✖')} ${err.message}`)
        return
      }
      console.error('Error writing to config file:', err);
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
