import { downloadMedia } from '@/lib/utils/handleMedia';
import { Api, TelegramClient } from 'telegram';
import { Raw } from 'telegram/events';
import terminalSize from 'term-size';
import terminalImage from 'terminal-image';
import { User } from '../lib/types/index';
import { ChatUser, FormattedMessage } from '../types';
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


const getOrganizedWebPageMedia = (media: MessageMediaWebPage): { url: string, displayUrl: string | null } => {
	return {
		url: media.webpage.url,
		displayUrl: 'displayUrl' in media.webpage ? media.webpage.displayUrl : null,
	}
}

const getOrganizedDocument = () => {
	//TODO: need to work on this
	// i need to figure how should i display this
	return {
		document: 'This file type is not supported by this Telegram client.'
	}
}

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
				const media = message.media as unknown as Media;

				const buffer =
					media && media.className == 'MessageMediaPhoto'
						? await downloadMedia({ media, size: 'large' })
						: null;

				const WebPage = media && media.className == 'MessageMediaWebPage' ? getOrganizedWebPageMedia(media as MessageMediaWebPage) : null;
				const width = (chatAreaWidth ?? terminalSize().columns * (70 / 100)) / 2;
				const document = media && media.className == 'MessageMediaDocument' ? getOrganizedDocument() : null
				const date = new Date(message.date * 1000);
				const imageString = await (buffer
					? terminalImage.buffer(new Uint8Array(buffer), {
						width
					})
					: null);


				return {
					id: message.id,
					sender: message.out ? 'you' : firstName,
					content: document ? "This Message is not supported by this Telegram client." : message.message,
					isFromMe: !!message.out,
					media: imageString,
					date,
					WebPage,
					document
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



interface BaseMedia {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: "MessageMediaWebPage" | "MessageMediaDocument" | "MessageMediaPhoto";
	classType: string;
	flags: number;
}

export type Media = MessageMediaWebPage | MessageMediaDocument | MessageMediaPhoto | null;

export interface MessageMediaWebPage extends BaseMedia {
	forceLargeMedia: boolean;
	forceSmallMedia: boolean;
	manual: boolean;
	safe: boolean;
	webpage: WebPage | WebPageEmpty;
}


export interface WebPage {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'WebPage';
	classType: string;
	flags: number;
	hasLargeMedia: boolean;
	id: bigint;
	url: string;
	displayUrl: string;
	hash: number;
	type: string;
	siteName: string;
	title: string;
	description: string | null;
	photo: Photo;
	embedUrl: string | null;
	embedType: string | null;
	embedWidth: number | null;
	embedHeight: number | null;
	duration: number | null;
	author: string | null;
	document: any | null;
	cachedPage: any | null;
	attributes: any | null;
}


export interface WebPageEmpty {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'WebPageEmpty';
	classType: string;
	flags: number;
	id: bigint;
	url: string;
}


export interface MessageMediaDocument extends BaseMedia {
	nopremium: boolean;
	spoiler: boolean;
	video: boolean;
	round: boolean;
	voice: boolean;
	document: Document;
	altDocuments: any | null;
	videoCover: any | null;
	videoTimestamp: any | null;
	ttlSeconds: number | null;
}

/**
 * Document interface.
 */
export interface Document {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'Document';
	classType: string;
	flags: number;
	id: bigint;
	accessHash: bigint;
	fileReference: Uint8Array;
	date: number;
	mimeType: string;
	size: bigint;
	thumbs: any | null;
	videoThumbs: any | null;
	dcId: number;
	attributes: any[];
}


export interface MessageMediaPhoto extends BaseMedia {
	spoiler: boolean;
	photo: Photo;
	ttlSeconds: number | null;
}


interface Photo {
	CONSTRUCTOR_ID: number;
	SUBCLASS_OF_ID: number;
	className: 'Photo';
	classType: string;
	flags: number;
	hasStickers: boolean;
	id: bigint;
	accessHash: bigint;
	fileReference: Uint8Array;
	date: number;
	sizes: any[];
	videoSizes: null;
	dcId: number;
}
