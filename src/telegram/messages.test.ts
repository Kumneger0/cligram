import { config } from 'dotenv';
import { TelegramClient } from 'telegram';
import { StringSession } from 'telegram/sessions/index.js';
import { afterAll, beforeAll, expect, it } from 'vitest';
import { deleteMessage, editMessage, sendMessage } from './messages';

config();

const apiId = process.env.TELEGRAM_API_ID;
const apiHash = process.env.TELEGRAM_API_HASH;

let client: TelegramClient;

beforeAll(async () => {
	if (!apiId || !apiHash) {
		throw new Error('API ID and API Hash are required');
	}
	client = new TelegramClient(
		new StringSession(process.env.SESSION_STRING),
		Number(apiId),
		apiHash!,
		{ connectionRetries: 5 }
	);
	await client.connect();
});

afterAll(async () => {
	await client.disconnect();
});

it('it should send a message', async () => {
	const me = await client.getMe();
	const result = await sendMessage(
		client,
		{ peerId: me.id, accessHash: me.accessHash! },
		'Hello World'
	);
	expect(result.messageId).toBeDefined();
});

it('it should send a message with reply', async () => {
	const me = await client.getMe();
	const message = await client.getMessages(me.id, { limit: 1 });
	const result = await sendMessage(
		client,
		{ peerId: me.id, accessHash: me.accessHash! },
		'Hello World',
		true,
		message[0]?.id
	);
	expect(result.messageId).toBeDefined();
});

it('it should edit a message', async () => {
	const me = await client.getMe();
	const result = await sendMessage(
		client,
		{ peerId: me.id, accessHash: me.accessHash! },
		'Hello World'
	);
	if (!result.messageId) {
		throw new Error('Message not sent');
	}
	const edited = await editMessage(
		client,
		{ peerId: me.id, accessHash: me.accessHash! },
		result.messageId,
		'Hello World Edited'
	);
	expect(edited).toBeDefined();
});

it('it should delete a message', async () => {
	const me = await client.getMe();
	const result = await sendMessage(
		client,
		{ peerId: me.id, accessHash: me.accessHash! },
		'Hello World'
	);
	if (!result.messageId) {
		throw new Error('Message not sent');
	}
	const deleted = await deleteMessage(
		client,
		{ peerId: me.id, accessHash: me.accessHash! },
		result.messageId
	);
	expect(deleted).toBeDefined();
});
