import { downloadMedia } from '@/lib/utils/handleMedia';
import { Api, TelegramClient } from 'telegram';
import { Raw } from 'telegram/events';
import terminalSize from 'term-size';
import terminalImage from 'terminal-image';
import {
	Dialog,
	FormattedMessage,
	Media,
	MessageMediaWebPage,
	TelegramUser
} from '../lib/types/index';
import { getUserInfo } from './client';
import { IterMessagesParams } from 'telegram/client/messages';

type ForwardMessageParams = {
	fromPeer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	id: number[];
	toPeer: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger };
	type: Dialog['peer']['className'];
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
 * @param {Dialog['peer']['className']} params.type - The type of the peer (e.g., 'PeerUser' or 'PeerChannel').
 * @returns {Promise<Api.messages.ForwardMessages>} The result of the forward operation.
 */
export async function forwardMessage(client: TelegramClient, params: ForwardMessageParams) {
	const fromPeerEntity =
		params.type === 'PeerUser'
			? new Api.InputPeerUser({
				userId: params.fromPeer.peerId,
				accessHash: params.fromPeer.accessHash
			})
			: new Api.InputPeerChannel({
				channelId: params.fromPeer.peerId,
				accessHash: params.fromPeer.accessHash
			});

	const toPeerEntity = new Api.InputPeerUser({
		userId: params.toPeer.peerId,
		accessHash: params.toPeer.accessHash
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
 * @param type - (Optional) The type of the peer to send the message to default is 'PeerUser'.
 * @returns An object containing the message ID and the result of the send operation.
 */
export const sendMessage = async (
	client: TelegramClient,
	peerInfo: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	message: string,
	isReply?: boolean | undefined,
	replyToMessageId?: number,
	type: Dialog['peer']['className'] = 'PeerUser'
) => {
	try {
		if (!client.connected) {
			await client.connect();
		}

		const sendMessageParam = {
			message: message,
			...(isReply && { replyTo: replyToMessageId })
		};
		const result = await client.sendMessage(
			type === 'PeerUser'
				? new Api.InputPeerUser({
					userId: peerInfo.peerId,
					accessHash: peerInfo.accessHash
				})
				: new Api.InputPeerChannel({
					channelId: peerInfo.peerId,
					accessHash: peerInfo.accessHash
				}),
			sendMessageParam
		);
		return {
			messageId: result.id,
			result
		};
	} catch (err) {
		return {
			messageId: null,
			result: null
		};
	}
};

/**
 * Deletes a message from a Telegram chat.
 *
 * @param client - The Telegram client instance.
 * @param peerInfo - An object containing the peer ID and access hash of the user to delete the message from.
 * @param messageId - The ID of the message to be deleted.
 * @param type - (Optional) The type of the peer to delete the message from default is 'PeerUser'.
 * @returns The result of the message deletion operation.
 */
export const deleteMessage = async (
	client: TelegramClient,
	peerInfo: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	messageId: number,
	type: Dialog['peer']['className'] = 'PeerUser'
) => {
	try {
		const result = await client.deleteMessages(
			type === 'PeerUser'
				? new Api.InputPeerUser({
					userId: peerInfo.peerId,
					accessHash: peerInfo.accessHash
				})
				: new Api.InputPeerChannel({
					channelId: peerInfo.peerId,
					accessHash: peerInfo.accessHash
				}),
			[Number(messageId)],
			{ revoke: true }
		);
		return result;
	} catch (err) {
		return null
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
	type: Dialog['peer']['className'] = 'PeerUser'
) => {
	try {
		const entity =
			type === 'PeerUser'
				? new Api.InputPeerUser({
					userId: peerInfo.peerId,
					accessHash: peerInfo.accessHash
				})
				: new Api.InputPeerChannel({
					channelId: peerInfo.peerId,
					accessHash: peerInfo.accessHash
				});
		const result = await client.invoke(
			new Api.messages.EditMessage({
				peer: entity,
				id: messageId,
				message: newMessage
			})
		);
		return result;
	} catch (err) {
		return null
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
export async function getAllMessages<T extends Dialog['peer']['className']>(
	{
		client,
		peerInfo: { accessHash, peerId: userId, userFirtNameOrChannelTitle },
		offsetId,
		chatAreaWidth
	}: {
		client: TelegramClient;
		peerInfo: {
			accessHash: bigInt.BigInteger | string;
			peerId: bigInt.BigInteger | string;
			userFirtNameOrChannelTitle: string;
		};
		offsetId?: number;
		chatAreaWidth?: number;
	},
	type: T,
	iterParams?: Partial<IterMessagesParams>
): Promise<FormattedMessage[]> {
	try {
		if (!client.connected) {
			await client.connect();
		}
		const messages = [];

		for await (const message of client.iterMessages(
			type === 'PeerUser'
				? new Api.InputPeerUser({
					userId: userId as unknown as bigInt.BigInteger,
					accessHash: accessHash as unknown as bigInt.BigInteger
				})
				: new Api.InputPeerChannel({
					channelId: userId as unknown as bigInt.BigInteger,
					accessHash: accessHash as unknown as bigInt.BigInteger
				}),
			{ limit: 10, offsetId, ...iterParams }
		)) {
			messages.push(message);
		}

		const orgnizedMessages = (
			await Promise.all(
				messages.reverse().map(async (message): Promise<FormattedMessage> => {
					const media = message.media as unknown as Media;

					// const buffer =
					// 	media && media.className === 'MessageMediaPhoto'
					// 		? await downloadMedia({ media, size: 'large' })
					// 		: null;

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
						document
					};
				})
			)
		)
			.map(({ content, ...rest }) => {
				return { content: content.trim(), ...rest };
			})
			.filter((msg) => {
				return msg.content.length > 0;
			});

		return orgnizedMessages;
	} catch (err) {
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
		onMessage: (message: FormattedMessage) => void;
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
		const user = (await getUserInfo(client, userId)) as unknown as TelegramUser;
		switch (event.className) {
			case 'UpdateShortMessage':
				onMessage({
					id: event.id,
					sender: event.out ? 'you' : user.firstName,
					content: event.message,
					isFromMe: event.out,
					media: null,
					date: event.date ? new Date(event.date * 1000) : new Date(),
					isUnsupportedMessage: false
				});
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
