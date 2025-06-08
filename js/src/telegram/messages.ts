import { Api, TelegramClient } from 'telegram';
import { IterMessagesParams, markAsRead } from 'telegram/client/messages';
import { Raw } from 'telegram/events';
import terminalSize from 'term-size';
import terminalImage from 'terminal-image';
import fs from 'node:fs/promises';
import {
	FormattedMessage,
	Media,
	MessageMediaWebPage,
	TelegramUser,
	ChatType,
	UserInfo
} from '../lib/types/index';
import { getUserInfo } from './client';
import { CustomFile } from 'telegram/client/uploads';

type GetEntityTypes = {
	peer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	type: ChatType;
};



const getEntity = ({ peer, type }: GetEntityTypes) => {
	const entity =
		type === 'user'
			? new Api.InputPeerUser({
				userId: peer.peerId,
				accessHash: peer.accessHash
			})
			: new Api.InputPeerChannel({
				channelId: peer.peerId,
				accessHash: peer.accessHash
			});
	return entity;
};

/**
 * Sets the typing status for a user in a chat on Telegram.
 *
 * @param {TelegramClient} client - The Telegram client instance
 * @param {Object} peer - The peer whose typing status should be set
 * @param {bigInt.BigInteger} peer.peerId - The ID of the peer
 * @param {bigInt.BigInteger} peer.accessHash - The access hash of the peer
 * @param {ChatType} type - The type of the peer (e.g., 'user' or 'channel')
 * @returns {Promise<void>} A promise that resolves when the typing status is set
 */
export const setUserTyping = async (
	client: TelegramClient,
	peer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	type: ChatType
) => {
	try {
		const entity = getEntity({ peer, type });
		await client.invoke(
			new Api.messages.SetTyping({
				peer: entity,
				action: new Api.SendMessageTypingAction()
			})
		);
	} catch (err) {
		console.error(err);
	}
};

type MarkUnReadParams = {
	client: TelegramClient;
	peer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	type: ChatType;
};

/**
 * Marks messages from a peer as unread on Telegram.
 *
 * @param {Object} params - The parameters for marking messages as unread
 * @param {TelegramClient} params.client - The Telegram client instance
 * @param {Object} params.peer - The peer whose messages should be marked as unread
 * @param {bigInt.BigInteger} params.peer.peerId - The ID of the peer
 * @param {bigInt.BigInteger} params.peer.accessHash - The access hash of the peer
 * @param {ChatType} params.type - The type of the peer (e.g., 'user' or 'channel')
 * @returns {Promise<any>} The result of marking messages as unread
 */
export const markUnRead = async ({ client, peer, type }: MarkUnReadParams) => {
	try {
		const entity = getEntity({ peer, type });
		const result = await markAsRead(client, entity);
		return result;
	} catch (err) {
		console.error(err);
	}
};

type ForwardMessageParams = {
	fromPeer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	id: number[];
	toPeer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	type: ChatType;
};

/**
 * Forwards messages from one peer to another on Telegram.
 *
 * @param {TelegramClient} client - The Telegram client instance.
 * @param {ForwardMessageParams} params - The parameters for forwarding the message.
 * @param {Object} params.fromPeer - The peer from which the message is forwarded.
 * @param {bigInt.BigInteger} params.fromPeer.peerId - The peer ID of the source.
 * @param {bigInt.BigInteger} params.fromPeer.accessHash - The access hash of the source.
 * @param {number[]} params.id - The IDs of the messages to forward.
 * @param {Object} params.toPeer - The peer to which the message is forwarded.
 * @param {bigInt.BigInteger} params.toPeer.peerId - The peer ID of the destination.
 * @param {bigInt.BigInteger} params.toPeer.accessHash - The access hash of the destination.
 * @param {ChatType} params.type - The type of the peer (e.g., 'user' or 'channel').
 * @returns {Promise<Api.messages.ForwardMessages>} The result of the forward operation.
 */
export async function forwardMessage(client: TelegramClient, params: ForwardMessageParams) {
	const fromPeerEntity = getEntity({
		peer: { accessHash: params.fromPeer.accessHash, peerId: params.fromPeer.peerId },
		type: params.type
	});
	const toPeerEntity = getEntity({
		peer: { accessHash: params.toPeer.accessHash, peerId: params.toPeer.peerId },
		//TODO: we need to allow forwarding to groups, own channel and bot let's hard this for now 
		type: "user"
	});

	const result = await client.invoke(
		new Api.messages.ForwardMessages({
			fromPeer: fromPeerEntity,
			id: params.id,
			toPeer: toPeerEntity
		})
	);
	return result;
}

/**
 * Sends a message to a Telegram user.
 *
 * @param client - The Telegram client instance.
 * @param peerInfo - An object containing the peer ID and access hash of the user to send the message to.
 * @param message - The message text to send.
 * @param isReply - (Optional) Indicates whether the message is a reply to another message.
 * @param replyToMessageId - (Optional) The ID of the message to reply to.
 * @param type - (Optional) The type of the peer to send the message to default is 'user'.
 * @param isFile - (optional) Indicates whether the message is a file.
 * @param path - (optional) - the path to file
 * @param onProgress - (optional) - a function to call on progress
 * @returns An object containing the message ID and the result of the send operation.
 */
export const sendMessage = async (
	client: TelegramClient,
	peerInfo: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	message: string,
	isReply?: boolean | undefined,
	replyToMessageId?: number,
	type: ChatType = 'user',
	isFile?: boolean | undefined,
	path?: string,
	onProgress?: (progress: number | null) => void
) => {
	if (!client.connected) {
		await client.connect();
	}
	const entityLike = type === 'group' ? peerInfo.peerId : getEntity({ peer: peerInfo, type });
	if (isFile && path) {
		const buffer = await fs.readFile(path);
		const fileName = path.split('/').pop() ?? 'file';
		const customeFile = new CustomFile(fileName, buffer.length, path);

		const toUpload = await client.uploadFile({
			file: customeFile,
			workers: 1,
			onProgress: (progress) => {
				if (onProgress) {
					onProgress(progress * 100);
				}
			}
		});

		const result = await client.sendFile(entityLike, {
			file: toUpload,
			caption: message,
			replyTo: isReply ? replyToMessageId : undefined
		});

		onProgress?.(null);

		return {
			messageId: result?.id,
		};
	}

	const sendMessageParam = {
		message: message,
		...(isReply && { replyTo: Number(replyToMessageId) })
	};
	const result = await client.sendMessage(entityLike, sendMessageParam);
	return {
		messageId: result?.id,
	};
};

/**
 * Deletes a message from a Telegram chat.
 *
 * @param client - The Telegram client instance.
 * @param peerInfo - An object containing the peer ID and access hash of the user to delete the message from.
 * @param messageId - The ID of the message to be deleted.
 * @param type - (Optional) The type of the peer to delete the message from default is 'user'.
 * @returns The result of the message deletion operation.
 */
export const deleteMessage = async (
	client: TelegramClient,
	peerInfo: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	messageId: number,
	type: ChatType = 'user'
) => {
	try {
		const entity = getEntity({ peer: peerInfo, type });
		const result = await client.deleteMessages(entity, [Number(messageId)], { revoke: true });
		return result;
	} catch (err) {
		return null;
	}
};

/**
 * Edits an existing message in a Telegram chat.
 *
 * @param client - The Telegram client instance.
 * @param peerInfo - An object containing the peer ID and access hash of the user to send the message to.
 * @param messageId - The ID of the message to be edited.
 * @param newMessage - The new message text to replace the existing message.
 * @returns The result of the message edit operation.
 */
export const editMessage = async (
	client: TelegramClient,
	peerInfo: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	messageId: number,
	newMessage: string,
	type: ChatType = 'user'
) => {
	try {
		const entity = getEntity({ peer: peerInfo, type });
		const result = await client.invoke(
			new Api.messages.EditMessage({
				peer: entity,
				id: messageId,
				message: newMessage
			})
		);
		return result;
	} catch (err) {
		return null;
	}
};

const getOrganizedWebPageMedia = (
	media: MessageMediaWebPage
): { url: string; displayUrl: string | null } => {
	return {
		url: media.webpage.url,
		displayUrl: 'displayUrl' in media.webpage ? media.webpage.displayUrl : null
	};
};

const getOrganizedDocument = () => {
	//TODO: need to work on this
	// i need to figure how should i display this
	return {
		document: 'This file type is not supported by this Telegram client.'
	};
};

/**
 * Retrieves all messages from a Telegram chat for the specified user or channel.
 *
 * @param client - The Telegram client instance.
 * @param peerInfo - An object containing the user's access hash, peer ID, and first name.
 * @param offsetId - The ID of the message to start retrieving messages from (optional).
 * @param chatAreaWidth - The width of the chat area (optional).
 * @returns An array of formatted messages.
 */
export async function getAllMessages<T extends ChatType>(
	client: TelegramClient,
	peerInfo: {
		accessHash: bigInt.BigInteger | string;
		peerId: bigInt.BigInteger | string;
		userFirtNameOrChannelTitle: string;
	},
	type: T,
	offsetId?: number,
	chatAreaWidth?: number,
	iterParams?: Partial<IterMessagesParams>
): Promise<FormattedMessage[]> {
	try {
		if (!client.connected) {
			await client.connect();
		}
		const { accessHash, peerId: userId, userFirtNameOrChannelTitle } = peerInfo

		const messages = [];
		const entity = getEntity({
			peer: { peerId: userId as bigInt.BigInteger, accessHash: accessHash as bigInt.BigInteger },
			type
		});
		const entityLike = type === 'group' ? userId : entity;
		for await (const message of client.iterMessages(entityLike, {
			limit: 50,
			offsetId,
			...iterParams
		})) {
			messages.push(message);
		}
		const orgnizedMessages = (
			await Promise.all(
				messages.reverse().map(async (message): Promise<FormattedMessage> => {
					const media = message.media as unknown as Media;

					const buffer = null;
					const webPage =
						media && media.className === 'MessageMediaWebPage'
							? getOrganizedWebPageMedia(media as MessageMediaWebPage)
							: null;
					const width = (chatAreaWidth ?? terminalSize().columns * (70 / 100)) / 2;
					const document =
						media && media.className === 'MessageMediaDocument' ? getOrganizedDocument() : null;
					const date = new Date(message.date * 1000);
					const imageString = await (buffer
						? terminalImage.buffer(new Uint8Array(buffer), {
							width
						})
						: null);

					return {
						isUnsupportedMessage: !!(media || document),
						id: message.id,
						sender: message.out ? 'you' : userFirtNameOrChannelTitle,
						content:
							document || media
								? 'This Message is not supported by this Telegram client.'
								: message.message,
						isFromMe: !!message.out,
						media: imageString,
						date,
						webPage,
						document,
						fromId:
							type === 'group' && 'fromId' in message
								? (message.fromId as { userId: bigInt.BigInteger }).userId
								: null
					};
				})
			)
		)
			.map(({ content, ...rest }) => {
				return { content: content?.trim(), ...rest };
			})
			.filter((msg) => {
				return msg?.content?.length > 0;
			});

		return orgnizedMessages;
	} catch (err) {
		const error = err as Error
		throw new Error(`${error.message} ${error.stack} ${error.name} ${error.cause}`)
		return [];
	}
}
/**
 * Listens for events from the Telegram client and handles them accordingly.
 *
 * @param {TelegramClient} client - The Telegram client instance to listen for events.
 * @param {Object} handlers - An object containing handler functions for different events.
 * @param {function(FormattedMessage): void} handlers.onMessage - A function to handle incoming messages.
 * @param {function(Object): void} [handlers.onUserOnlineStatus] - A function to handle user online status updates.
 *
 * @returns {function(): void} A function to remove the event handler when called.
 */
export const listenForEvents = async (
	client: TelegramClient,
	{
		onMessage,
		onUserOnlineStatus
	}: {
		onMessage: (message: FormattedMessage, user: Omit<UserInfo, 'unreadCount'> | null) => void;
		onUserOnlineStatus?: (user: {
			accessHash: string;
			firstName: string;
			status: 'online' | 'offline';
			lastSeen?: number;
		}) => void;
	}
) => {
	if (!client.connected) {
		await client.connect();
	}

	type Event = {
		date: number;
		userId: bigInt.BigInteger;
		className: string;
		id: number;
		message: string;
		out: boolean;
		status: {
			className: string;
		};
	};
	const hanlder = async (event: Event) => {
		const userId = event.userId;
		const user = await getUserInfo(client, userId);

		if (!user) {
			return;
		}

		switch (event.className) {
			case 'UpdateShortMessage':
				console.log(event);
				onMessage(
					{
						id: event.id,
						sender: event.out ? 'you' : user.firstName,
						content: event.message,
						isFromMe: event.out,
						media: null,
						date: event.date ? new Date(event.date * 1000) : new Date(),
						isUnsupportedMessage: false
					},
					user
				);
				break;
			case 'UpdateUserStatus':
				if (event.status.className === 'UserStatusOnline') {
					onUserOnlineStatus &&
						onUserOnlineStatus({
							accessHash: user.accessHash.toString(),
							firstName: user.firstName,
							status: 'online'
						});
				}
				if (event.status.className === 'UserStatusOffline') {
					const user = (await client.getEntity(
						await client.getInputEntity(userId)
					)) as unknown as TelegramUser | null;
					if (!user) {
						return;
					}
					onUserOnlineStatus &&
						onUserOnlineStatus({
							accessHash: user.accessHash.toString(),
							firstName: user.firstName,
							status: 'offline',
							lastSeen: user.status?.wasOnline
						});
				}
				break;
			default:
				break;
		}
	};

	client.addEventHandler(hanlder);
	return () => {
		const event = new Raw({});
		return client.removeEventHandler(hanlder, event);
	};
};
