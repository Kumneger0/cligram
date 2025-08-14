#!/usr/bin/env node
import { Buffer } from 'buffer';
import { stderr, stdin, stdout } from 'process';

import { TelegramClient } from 'telegram';
import { RPCError as TelegramRpcError } from 'telegram/errors/index.js';
import { LogLevel } from 'telegram/extensions/Logger.js';
import { login, logout } from './commands';
import { FormattedMessage, UserInfo } from './lib/types';
import { logger } from './lib/utils';
import { getTelegramClient } from './lib/utils/auth';
import { getUserChats, getUserInfo, searchUsers } from './telegram/client';
import { deleteMessage, editMessage, forwardMessage, getAllMessages, listenForEvents, markUnRead, phoneCall, sendMessage, setUserTyping } from './telegram/messages';

export const callKey = new Map<"privateKey" | "prime-modulus" | "publicKeyBytes", bigInt.BigInteger | Buffer>()

const stringify = JSON.stringify

/**
 * gram.js has logs that we don't need i tried setting log level to none but it didn't work
 * so we just patch the global console object to ignore all logs
 */


for (const mName of Object.keys(console)) {
	console[mName] = () => {
		// do nothing 
	}
}

const arg = process.argv[2]
type Handler = (client: TelegramClient, ...args: unknown[]) => Promise<unknown>;
const handlers: Record<string, Handler> = {
	sendMessage,
	deleteMessage,
	editMessage,
	searchUsers,
	getUserChats,
	getUserInfo,
	getAllMessages,
	forwardMessage,
	markUnRead,
	setUserTyping,
	phoneCall
} as const


type RestParameters<TFunc extends (client: any, ...args: any[]) => any> = TFunc extends (
	client: TelegramClient,
	...args: infer P
) => any
	? P
	: never;

type MethodParamsMap = {
	[K in keyof typeof handlers]: RestParameters<(typeof handlers)[K]>;
};

type TypedRpcRequest = {
	[M in keyof MethodParamsMap]: {
		jsonrpc: '2.0';
		id: number;
		method: M;
		params: MethodParamsMap[M];
	};
}[keyof MethodParamsMap];

type TypedRpcNotification = {
	[M in keyof MethodParamsMap]: {
		jsonrpc: '2.0';
		method: M;
		params: MethodParamsMap[M];
	};
}[keyof MethodParamsMap];

type RpcSuccess<Response = unknown> = {
	jsonrpc: '2.0';
	id: number;
	result: Response;
};

type RpcErrorResponse = {
	jsonrpc: '2.0';
	id: number | null;
	error: {
		code: number;
		message: string;
		data?: unknown;
	};
};


type IncomingMessage = TypedRpcRequest | TypedRpcNotification;

type NewMessageParams = {
	message: FormattedMessage,
	user: UserInfo
}
type UserTypingParams = {
	user: UserInfo
}

type UserOnlineOfflineParams = {
	accessHash: string, firstName: string, status: 'online' | 'offline', lastSeen?: Date
}

type RpcTelegramEventsNotification = {
	jsonrpc: '2.0';
} & ({
	method: "newMessage";
	params: NewMessageParams;
} | {
	method: "userOnlineOffline";
	params: UserOnlineOfflineParams;
} | {
	method: 'userTyping',
	params: UserTypingParams
})


async function readHeaders(reader: typeof stdin): Promise<{ [key: string]: string }> {
	const headers: { [key: string]: string } = {};
	let lineBuffer = '';

	while (true) {
		const chunk = reader.read(1);
		if (chunk === null) {
			await new Promise((resolve) => reader.once('readable', resolve));
			continue;
		}
		const char = chunk.toString('utf8');
		lineBuffer += char;

		if (lineBuffer.endsWith('\r\n') || lineBuffer.endsWith('\n')) {
			let line: string;
			if (lineBuffer.endsWith('\r\n')) {
				line = lineBuffer.slice(0, -2);
			} else {
				line = lineBuffer.slice(0, -1);
			}
			if (line === '') {
				break;
			}
			const parts = line.split(':');
			if (parts.length >= 2) {
				const headerName = parts[0]!.trim();
				const headerValue = parts[1].trim();
				headers[headerName] = headerValue;
			}
			lineBuffer = '';
		}
	}
	return headers;
}

async function readMessage(): Promise<IncomingMessage> {
	const headers = await readHeaders(stdin);
	const contentLengthHeader = Object.keys(headers).find(
		(h) => h.toLowerCase() === 'content-length'
	);

	if (!contentLengthHeader) {
		stderr.write('Error: Missing Content-Length header\n' + stringify(headers) + '\n');
		throw new Error('Missing Content-Length header');
	}
	const length = parseInt(headers[contentLengthHeader!]!, 10);
	if (isNaN(length) || length <= 0) {
		stderr.write('Error: Invalid Content-Length header: ' + headers[contentLengthHeader] + '\n');
		throw new Error('Invalid Content-Length header');
	}

	let payloadBuffer = Buffer.alloc(length);
	let bytesRead = 0;
	while (bytesRead < length) {
		const chunk = stdin.read(length - bytesRead);
		if (chunk === null) {
			await new Promise((resolve) => stdin.once('readable', resolve));
			continue;
		}
		chunk.copy(payloadBuffer, bytesRead);
		bytesRead += chunk.length;
	}

	const payload = payloadBuffer.toString('utf8', 0, length);

	try {
		return JSON.parse(payload) as IncomingMessage;
	} catch (e) {
		stderr.write('Error: Failed to parse JSON payload: ' + payload + '\n' + e + '\n');
		throw new Error('Parse error');
	}
}

function writeToStdout(msg: RpcSuccess | RpcErrorResponse | RpcTelegramEventsNotification): void {
	const json = stringify(msg);
	const header = `Content-Length: ${Buffer.byteLength(json, 'utf8')}\r\n\r\n`;
	stdout.write(header + json);
}

function createRpcError(
	id: number | null,
	code: number,
	message: string,
	data?: unknown
): RpcErrorResponse {
	return {
		jsonrpc: '2.0',
		id,
		error: { code, message, data }
	};
}

let telegramClientInstance: TelegramClient | null = null;


let cleanUp: () => void;

async function startup() {
	logger.info('Starting up the application');
	try {
		if (arg === "login") {
			await login()
			return
		}

		if (arg === 'logout') {
			await logout()
			return
		}

		const client = await getTelegramClient();
		if (!client.connected) await client.connect()

		if (client) {
			telegramClientInstance = client;
			await telegramClientInstance.connect();
			const me = await telegramClientInstance.getMe();
			if (typeof me !== 'boolean' && me?.phone) {
				telegramClientInstance.setLogLevel(LogLevel.NONE);
				//TODO: this is making the app to freeze 
				// fix this
				// await setUserPrivacy(telegramClientInstance);
			} else if (typeof me !== 'boolean' && !me?.phone) {
				telegramClientInstance.setLogLevel(LogLevel.NONE);
			} else {
			}
		} else {
			writeToStdout(createRpcError(null, -32323, 'Failed to connect to telegram'));
		}
	} catch (err: any) {
		let message = 'Initialization failed.';
		let code = -32000;

		if (err instanceof TelegramRpcError) {
			message = err.message;
			if (err.code) code = err.code;
		} else if (err instanceof Error) {
			message = err.message;
		}

		writeToStdout(createRpcError(null, code, message));
		process.exit(1);
	}

	if (!telegramClientInstance) {
		createRpcError(null, -3200, 'Failed to get Telegram client instace');
		process.exit(1);
	}
	await messageProcessingLoop(telegramClientInstance);
}

async function messageProcessingLoop(client: TelegramClient) {
	cleanUp = await listenForEvents(client, {
		updateUserTyping(user) {
			const userTypingEvent: RpcTelegramEventsNotification = {
				jsonrpc: '2.0',
				method: 'userTyping',
				params: {
					user: {
						...user,
						isOnline: true,
						isTyping: true
					}
				}
			}
			writeToStdout(userTypingEvent)
		},
		onMessage(message, user) {
			const telegramMessageEvent: RpcTelegramEventsNotification = {
				jsonrpc: '2.0',
				method: 'newMessage',
				params: {
					message,
					user
				}
			}
			writeToStdout(telegramMessageEvent)
		},
		onUserOnlineStatus(user) {
			const telegramUserOnlineEvent: RpcTelegramEventsNotification = {
				jsonrpc: '2.0',
				method: 'userOnlineOffline',
				params: {
					...user,
					lastSeen: user.lastSeen ? new Date(user.lastSeen * 1000) : undefined
				}
			}
			writeToStdout(telegramUserOnlineEvent)
		}
	});

	while (true) {
		let msg: IncomingMessage;
		try {
			msg = await readMessage();
		} catch (err: any) {
			if (err.message === 'Parse error') {
				writeToStdout(
					createRpcError(null, -32700, 'Parse error: Invalid JSON was received by the server.')
				);
			} else {
				stderr.write(`Read error: ${err.message || err}\n`);
				writeToStdout(
					createRpcError(null, -32603, `Internal error: ${err.message || 'Failed to read message'}`)
				);
			}
			continue;
		}

		if (!msg || typeof msg.method !== 'string') {
			const IncomingMessage = msg as Object;
			writeToStdout(
				createRpcError(
					'id' in IncomingMessage && typeof IncomingMessage.id === 'number'
						? IncomingMessage.id
						: null,
					-32600,
					'Invalid Request: Malformed message.'
				)
			);
			continue;
		}

		if ('id' in msg && typeof msg.id === 'number') {
			const request = msg as TypedRpcRequest;

			try {
				let result: unknown;
				try {
					const method = handlers[request.method]
					result = await method(client, ...request.params)
				} catch (error) {
					writeToStdout(
						createRpcError(
							Number((request as { id: number }).id),
							-32601,
							`Error handling method ${request.method}: ${error.message || 'Unknown error'}`
						)
					);
					continue;
				}
				const response: RpcSuccess = { jsonrpc: '2.0', id: request.id, result };
				writeToStdout(response);
			} catch (error) {
				let errorCode = -32000;
				let errorMessage = 'An unexpected error occurred in the handler.';
				let errorData: unknown | undefined;

				if (error instanceof TelegramRpcError) {
					errorMessage = error.message;
					if (error.code) errorCode = error.code;
				} else if (error instanceof Error) {
					errorMessage = error.message;
				} else if (typeof error === 'string') {
					errorMessage = error;
				}
				logger.error(error)
				writeToStdout(createRpcError(request.id, errorCode, errorMessage, errorData));
			}
		} else {
			const notification = msg as TypedRpcNotification;
			if (typeof handlers[notification.method] === 'function') {
				try {
					await handlers[notification.method]?.(client, ...notification.params)
				} catch (error) {
					logger.error(error)
					stderr.write(
						`Error in notification handler for ${notification.method}: ${error.message || error}\n`
					);
				}
			} else {
				stderr.write(`Received notification for unknown method: ${notification.method}\n`);
			}
		}
	}
}

async function shutdown(signal: string) {
	stderr.write(`\nReceived ${signal}. Shutting down...\n`);
	if (telegramClientInstance && telegramClientInstance.connected) {
		try {
			stderr.write('Disconnecting Telegram client...\n');
			await telegramClientInstance.disconnect();
			stderr.write('Telegram client disconnected.\n');
		} catch (e) {
			if (e instanceof Error) {
				logger.error(e)
				stderr.write(`Error during client disconnect: ${e.message}\n`);
			}
		}
	}
	cleanUp?.()
	process.exit(0);
}

process.on('SIGINT', () => shutdown('SIGINT'));
process.on('SIGTERM', () => shutdown('SIGTERM'));

try {
	await startup().catch((err) => {
		stderr.write(`Fatal error in startup promise: ${err.message || err}\nStack: ${err?.stack}\n`);
	});
} catch (err) {
	logger.error(err)
}
