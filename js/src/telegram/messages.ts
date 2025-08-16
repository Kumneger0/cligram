import { getConfig } from '@/config/configManager';
import { entityCache, logger, sendSystemNotification } from '@/lib/utils';
import bigInt from 'big-integer';
import notifier from 'node-notifier';
import crypto from 'node:crypto';
import fs from 'node:fs/promises';
import { Api, TelegramClient } from 'telegram';
import { IterMessagesParams, markAsRead } from 'telegram/client/messages';
import { CustomFile } from 'telegram/client/uploads';
import { Raw } from 'telegram/events';
import {
	generateRandomBigInt,
	readBigIntFromBuffer,
	readBufferFromBigInt,
	sha256
} from 'telegram/Helpers';
import terminalSize from 'term-size';
import terminalImage from 'terminal-image';

import { callKey } from '..';
import {
	Channel,
	ChannelInfo,
	ChatType,
	FormattedMessage,
	Media,
	MessageMediaWebPage,
	TelegramUser,
	UserInfo
} from '../lib/types/index';
import { getUserInfo } from './client';

type GetEntityTypes = {
	peer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
};

const getEntity = async ({ peer }: GetEntityTypes, client: TelegramClient) => {
	const peerIdString = peer.peerId.toString();
	if (entityCache.has(peerIdString)) {
		const entity = entityCache.get(peerIdString);
		return entity as Api.TypeInputPeer;
	}
	const entity = await client.getInputEntity(peer.peerId);
	entityCache.set(peerIdString, entity);
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
		const entity = await getEntity({ peer }, client);
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
export const markUnRead = async (
	client: TelegramClient,
	peer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger }
) => {
	try {
		const entity = await getEntity({ peer }, client);
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
	const fromPeerEntity = await getEntity(
		{
			peer: { accessHash: params.fromPeer.accessHash, peerId: params.fromPeer.peerId }
		},
		client
	);
	const toPeerEntity = await getEntity(
		{
			peer: { accessHash: params.toPeer.accessHash, peerId: params.toPeer.peerId }
		},
		client
	);

	const result = await client.invoke(
		new Api.messages.ForwardMessages({
			fromPeer: fromPeerEntity,
			id: params.id,
			toPeer: toPeerEntity
		})
	);
	return result;
}

interface PeerInfo {
	peerId: bigInt.BigInteger;
	accessHash: bigInt.BigInteger;
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
	peerInfo: PeerInfo,
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
	const entityLike =
		type === 'group' ? peerInfo.peerId : await getEntity({ peer: peerInfo }, client);
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
			messageId: result?.id
		};
	}

	const sendMessageParam = {
		message: message,
		...(isReply && { replyTo: Number(replyToMessageId) })
	};
	const result = await client.sendMessage(entityLike, sendMessageParam);
	return {
		messageId: result?.id
	};
};

const PROTOCOL_LAYERS = {
	minLayer: 93,
	maxLayer: 93
};

export const phoneCall = async (client: TelegramClient, calleeEntity: Api.TypeInputPeer) => {
	const dhConfig = await client.invoke(
		new Api.messages.GetDhConfig({
			version: 0,
			randomLength: 256
		})
	);
	if (dhConfig instanceof Api.messages.DhConfigNotModified) {
		throw new Error('Invalid DHConfig');
	}

	const primeP = readBigIntFromBuffer(dhConfig.p, false, false);
	const generatorG = bigInt(dhConfig.g);

	let privateKeyA = bigInt.one;
	while (!(bigInt.one.lesser(privateKeyA) && privateKeyA.lesser(primeP.minus(1)))) {
		privateKeyA = generateRandomBigInt();
	}

	const publicKeyGA = generatorG.modPow(privateKeyA, primeP);
	const publicKeyGABytes = readBufferFromBigInt(publicKeyGA, 256, false, false);
	const publicKeyGAHash = await sha256(publicKeyGABytes);

	await client.invoke(
		new Api.phone.RequestCall({
			userId: calleeEntity,
			gAHash: publicKeyGAHash,
			randomId: bigInt.randBetween('1', '2147483647').toJSNumber(),
			protocol: new Api.PhoneCallProtocol({
				minLayer: 65,
				maxLayer: 139,
				udpP2p: true,
				udpReflector: true,
				libraryVersions: ['4.0.0', '3.0.0', '2.7.7', '2.4.4']
			})
		})
	);

	callKey.set('prime-modulus', primeP);
	callKey.set('privateKey', privateKeyA);
	callKey.set('publicKeyBytes', publicKeyGABytes);
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
	messageId: number
) => {
	const entity = await getEntity({ peer: peerInfo }, client);
	await client.deleteMessages(entity, [Number(messageId)], { revoke: true });
	return {
		status: 'success'
	};
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
	newMessage: string
) => {
	const entity = await getEntity({ peer: peerInfo }, client);
	await client.invoke(
		new Api.messages.EditMessage({
			peer: entity,
			id: messageId,
			message: newMessage
		})
	);
	return true;
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
		const { accessHash, peerId: userId, userFirtNameOrChannelTitle } = peerInfo;

		const messages: Api.Message[] = [];
		const entity = await getEntity(
			{
				peer: { peerId: userId as bigInt.BigInteger, accessHash: accessHash as bigInt.BigInteger }
			},
			client
		);
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

					const senderPeerId =
						type === 'group' && 'fromId' in message
							? (message?.fromId as { userId: bigInt.BigInteger })?.userId
							: null;

					let senderUserInfo = senderPeerId ? await getUserInfo(client, senderPeerId) : null;

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
						fromId: senderPeerId,
						senderUserInfo
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
		const error = err as Error;
		throw new Error(`${error.message} ${error.stack} ${error.name} ${error.cause}`);
	}
}

function bigIntToBuffer(num: bigInt.BigInteger, length: number) {
	let hex = num.toString(16);
	if (hex.length % 2) hex = '0' + hex;
	const buf = Buffer.from(hex, 'hex');
	return Buffer.concat([Buffer.alloc(length - buf.length), buf]);
}

export const handlePhoneCallAccepted = async (
	client: TelegramClient,
	call: Api.PhoneCallAccepted
) => {
	try {
		//TODO: need to work this
		const myPrivateA = callKey.get('privateKey') as bigInt.BigInteger;
		const primeP = callKey.get('prime-modulus') as bigInt.BigInteger;
		const myGABytes = callKey.get('publicKeyBytes') as Buffer;

		const gB = readBigIntFromBuffer(call.gB, false, false);
		const sharedKey = gB.modPow(myPrivateA, primeP);
		const sharedKeyBytes = bigIntToBuffer(sharedKey, 256);

		const keyFingerprint = crypto.createHash('sha1').update(sharedKeyBytes).digest().slice(-8);

		const inputPhoneCall = new Api.InputPhoneCall({
			id: call.id,
			accessHash: call.accessHash
		});

		await client.invoke(
			new Api.phone.ConfirmCall({
				peer: inputPhoneCall,
				gA: myGABytes,
				keyFingerprint: BigInt(
					'0x' + keyFingerprint.toString('hex')
				) as unknown as bigInt.BigInteger,
				protocol: new Api.PhoneCallProtocol({
					...PROTOCOL_LAYERS,
					udpP2p: true,
					udpReflector: true,
					libraryVersions: ['4.0.0', '3.0.0', '2.7.7', '2.4.4']
				})
			})
		);
	} catch (err) {
		logger.error(err);
	}
};

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

const handlePhoneCallEvent = async (client: TelegramClient, event: Event) => {
	if (event.className === 'UpdatePhoneCall') {
		const call = (event as unknown as { phoneCall: Api.PhoneCallAccepted }).phoneCall;
		if (call.className !== 'PhoneCallAccepted') {
			return;
		}
		await handlePhoneCallAccepted(client, call);
	}
};

const handleTypingEvent = async (
	client: TelegramClient,
	event: Event,
	cb: (u: UserInfo) => void
) => {
	if (event.className === 'UpdateUserTyping') {
		const user = await getUserInfo(client, event.userId);
		if (user) {
			cb(user);
		}
	}
};

const handleMessageEvent = async (
	client: TelegramClient,
	event: Event,
	onMessage: (onNewMessage: onMessageArg) => void
) => {
	if (event.className === 'UpdateShortMessage') {
		const user = await getUserInfo(client, event.userId);
		if (user) {
			const config = getConfig('notifications');
			if (config?.enabled && !event.out) {
				sendSystemNotification({
					title: `Cligram - ${user.firstName} sent you a message`,
					message: config.showMessagePreview ? event.message : '',
					icon: 'https://telegram.org/favicon.ico'
				});
			}
			onMessage({
				message: {
					id: event.id,
					sender: event.out ? 'you' : user.firstName,
					content: event.message,
					isFromMe: event.out,
					media: null,
					date: event.date ? new Date(event.date * 1000) : new Date(),
					isUnsupportedMessage: false
				},
				user
			});
		}
	}
};

type OnUserOnlineStatus = (user: {
	accessHash: string;
	firstName: string;
	status: 'online' | 'offline';
	lastSeen?: number;
}) => void;

const handleUserStatusEvent = async (
	client: TelegramClient,
	event: Event,
	onUserOnlineStatus: OnUserOnlineStatus
) => {
	if (event.className === 'UpdateUserStatus') {
		const user = await getUserInfo(client, event.userId);
		if (user) {
			if (event.status.className === 'UserStatusOnline') {
				onUserOnlineStatus?.({
					accessHash: user.accessHash.toString(),
					firstName: user.firstName,
					status: 'online'
				});
			}
			if (event.status.className === 'UserStatusOffline') {
				const userEntity = (await client.getEntity(
					await client.getInputEntity(event.userId)
				)) as unknown as TelegramUser | null;
				if (userEntity) {
					onUserOnlineStatus?.({
						accessHash: userEntity.accessHash.toString(),
						firstName: userEntity.firstName,
						status: 'offline',
						lastSeen: userEntity.status?.wasOnline
					});
				}
			}
		}
	}
};

type onMessageArg =
	| {
			message: FormattedMessage;
			user: UserInfo;
	  }
	| {
			message: FormattedMessage;
			channelOrGroup: ChannelInfo;
	  };

export const listenForEvents = async (
	client: TelegramClient,
	{
		onMessage,
		onUserOnlineStatus,
		updateUserTyping
	}: {
		onMessage: (onNewMessage: onMessageArg) => void;
		updateUserTyping: (user: UserInfo) => void;
		onUserOnlineStatus?: OnUserOnlineStatus;
	}
) => {
	if (!client.connected) {
		await client.connect();
	}

	const eventHandler = async (event: Event) => {
		await handlePhoneCallEvent(client, event);
		await handleTypingEvent(client, event, updateUserTyping);
		await handleMessageEvent(client, event, onMessage);
		await handleUserStatusEvent(client, event, onUserOnlineStatus);
		await updateShortChatMessage(client, event, onMessage);
		await updateNewChannelMessage(client, event, onMessage);
	};

	client.addEventHandler(eventHandler);
	return () => {
		const event = new Raw({});
		return client.removeEventHandler(eventHandler, event);
	};
};

async function updateNewChannelMessage(
	client: TelegramClient,
	event: Event,
	onMessage: (onNewMessage: onMessageArg) => void
) {
	if (event.className == 'UpdateNewChannelMessage') {
		const {
			peerId: { channelId },
			message,
			id
		} = event.message as unknown as Event & {
			peerId: { channelId: string };
			id: number;
			message: string;
		};
		const entity = (await client.getEntity(channelId)) as unknown as Channel;
		const config = getConfig('notifications');
		if (config.enabled) {
			sendSystemNotification({
				title: `new message in ${entity.title}`,
				message: config.showMessagePreview ? message : '',
				icon: 'https://telegram.org/favicon.ico'
			});
		}
		const channel = {
			accessHash: '',
			channelId: channelId,
			isBroadcast: true,
			isCreator: false,
			participantsCount: entity.participantsCount,
			title: entity.title,
			unreadCount: 0, // let's ignore this for now since this is for notification most of time this info will be avaialbe before
			username: entity.username
		} satisfies ChannelInfo;
		onMessage({
			channelOrGroup: channel,
			message: {
				content: message,
				sender: entity.title,
				date:
					'time' in (event.message as unknown as Record<string, unknown>)
						? new Date(
								((event.message as unknown as Record<string, unknown>).time as number) * 1000
							)
						: new Date(),
				id,
				isUnsupportedMessage: false,
				isFromMe: false,
				media: null
			}
		});
	}
}

async function updateShortChatMessage(
	client: TelegramClient,
	event: Event,
	onMessage: (onNewMessage: onMessageArg) => void
) {
	if (event.className == 'UpdateShortChatMessage') {
		const { chatId, fromId } = event as Event & {
			chatId: bigInt.BigInteger;
			fromId: bigInt.BigInteger;
		};
		const entity = (await client.getEntity(chatId)) as unknown as Channel;
		const user = await getUserInfo(client, fromId);
		const config = getConfig('notifications');
		if (config.enabled) {
			sendSystemNotification({
				title: `${user.firstName} sent new message in ${entity.title}`,
				message: config.showMessagePreview ? event.message : '',
				icon: 'https://telegram.org/favicon.ico'
			});
		}
		const group = {
			title: entity.title,
			username: '',
			channelId: chatId.toString(),
			accessHash: String(entity.accessHash),
			isCreator: entity?.creator ?? false,
			isBroadcast: false,
			participantsCount: entity.participantsCount,
			unreadCount: 0 // let's ignore this for now since this is for notification most of time this info will be avaialbe before
		} satisfies ChannelInfo;

		onMessage({
			message: {
				id: event.id,
				sender: event.out ? 'you' : user.firstName,
				content: event.message,
				isFromMe: event.out,
				media: null,
				date: event.date ? new Date(event.date * 1000) : new Date(),
				isUnsupportedMessage: false
			},
			channelOrGroup: group
		});
	}
}
