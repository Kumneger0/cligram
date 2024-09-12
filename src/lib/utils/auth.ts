import { TelegramClient } from "telegram";
import { StringSession } from "telegram/sessions";
import input from "input";
import os from "node:os";

import { config } from "dotenv";

import path from "node:path";
import fs from "node:fs";

config();

const apiId = process.env.TELEGRAM_API_ID;
const apiHash = process.env.TELEGRAM_API_HASH;

export const session = getUserSession();

const stringSession = new StringSession(session ?? "");

export async function getTelegramClient() {
  const client = new TelegramClient(stringSession, Number(apiId), apiHash!, {
    connectionRetries: 5,
  });

  if (session) return client;

  await client.start({
    phoneNumber: async () => {
      const phoneNumber = await input.text("Please enter your number: ");
      return phoneNumber;
    },
    password: async () => {
      const password = await input.text("Please enter your password: ");
      return password;
    },
    phoneCode: async () => {
      const phoneCode = await input.text(
        "Please enter the code you received: "
      );
      return phoneCode;
    },
    onError: (err) => console.error("Authentication error:", err),
  });
  console.log("You should now be connected.");
  const telegramSession = client.session.save();

  const configData = `session=${telegramSession}`;

  setUserConfigration(configData);
  return client;
}

function setUserConfigration(configData: string) {
  const homeDir = os.homedir();

  const configDir = path.join(homeDir, ".tg-cli");
  const configFile = path.join(configDir, "config.txt");

  if (!fs.existsSync(configDir)) {
    fs.mkdirSync(configDir, { recursive: true });
  }

  fs.writeFile(configFile, configData, (err) => {
    if (err) {
      console.error("Error writing to config file:", err);
    } else {
      console.log("Configuration file created successfully at", configFile);
    }
  });
}

function getUserSession() {
  const homeDir = os.homedir();
  const configDir = path.join(homeDir, ".tg-cli");
  const configFile = path.join(configDir, "config.txt");
  if (!fs.existsSync(configFile)) {
    return null;
  }
  const [key, value] = fs.readFileSync(configFile, "utf-8").split("=");
  if (!key || key !== "session") {
    return null;
  }
  return value;
}
