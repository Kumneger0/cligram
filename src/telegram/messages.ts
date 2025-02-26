import { downloadMedia } from '@/lib/utils/handleMedia';
import { Api, TelegramClient } from 'telegram';
import { Raw } from 'telegram/events';
import terminalSize from 'term-size';
import terminalImage from 'terminal-image';
import { User } from '../lib/types/index';
import { ChatUser, FormattedMessage, MessageMedia } from '../types';
import { getUserInfo } from './client';

export const sendMessage = async (
	client: TelegramClient,
	userToSend: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	message: string,
	isReply?: boolean | undefined,
	replyToMessageId?: number
) => {
	if (!client.connected) await client.connect();

	const sendMessageParam = {
		message: message,
		...(isReply && { replyTo: replyToMessageId })
	};
	await client.sendMessage(
		new Api.InputPeerUser({
			userId: userToSend?.peerId,
			accessHash: userToSend?.accessHash
		}),
		sendMessageParam
	);
};

export const deleteMessage = async (
	client: TelegramClient,
	userToSend: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	messageId: number
) => {
	try {
		const result = await client.deleteMessages(
			new Api.InputPeerUser({
				userId: userToSend?.peerId,
				accessHash: userToSend?.accessHash
			}),
			[Number(messageId)],
			{ revoke: true }
		);
		return result;
	} catch (err) {
		console.log(err);
	}
};

export const editMessage = async (
	client: TelegramClient,
	userToSend: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	messageId: number,
	newMessage: string
) => {
	try {
		const entity = new Api.InputPeerUser({
			userId: userToSend?.peerId,
			accessHash: userToSend?.accessHash
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
		console.log(err);
	}
};

export async function getAllMessages({
	client,
	user: { accessHash, peerId: userId, firstName },
	offsetId,
	chatAreaWidth
}: {
	client: TelegramClient;
	user: ChatUser;
	offsetId?: number;
	chatAreaWidth?: number;
}): Promise<FormattedMessage[]> {
	if (!client.connected) await client.connect();
	const messages = [];

	for await (const message of client.iterMessages(
		new Api.InputPeerUser({
			userId: userId as unknown as bigInt.BigInteger,
			accessHash
		}),
		{ limit: 20, offsetId }
	)) {
		messages.push(message);
	}

	const orgnizedMessages = (
		await Promise.all(
			messages.reverse()?.map(async (message): Promise<FormattedMessage> => {
				const media = message.media as unknown as MessageMedia;
				const buffer =
					media && media.className == 'MessageMediaPhoto'
						? await downloadMedia({ media, size: 'large' })
						: null;

				const width = (chatAreaWidth ?? terminalSize().columns * (70 / 100)) / 2;

				const date = new Date(message.date * 1000);

				const imageString = await (buffer
					? terminalImage.buffer(new Uint8Array(buffer), {
						width
					})
					: null);
				return {
					id: message.id,
					sender: message.out ? 'you' : firstName,
					content: message.message,
					isFromMe: !!message.out,
					media: imageString,
					date
				};
			})
		)
	)
		?.map(({ content, ...rest }) => ({ content: content?.trim(), ...rest }))
		?.filter((msg) => msg?.content?.length > 0);

	return orgnizedMessages;
}

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
	if (!client.connected) await client.connect();

	interface Event {
		date: number;
		userId: bigInt.BigInteger;
		className: string;
		id: number;
		message: string;
		out: boolean;
		status: {
			className: string;
		};
	}
	const hanlder = async (event: Event) => {
		const userId = event.userId;
		if (userId) {
			const user = (await getUserInfo(client, userId)) as unknown as User;
			switch (event.className) {
				case 'UpdateShortMessage':
					onMessage &&
						onMessage({
							id: event.id,
							sender: event.out ? 'you' : user.firstName,
							content: event.message,
							isFromMe: event.out,
							media: null,
							date: event.date ? new Date(event.date * 1000) : new Date()
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
					console.log('unknown event', event);
					break;
			}
		}
	};

	client.addEventHandler(hanlder);
	return () => {
		const event = new Raw({});
		return client.removeEventHandler(hanlder, event);
	};
};
