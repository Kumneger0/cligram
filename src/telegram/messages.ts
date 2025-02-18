import { downloadMedia } from '@/lib/utils/handleMedia';
import { Api, TelegramClient } from 'telegram';
import { User } from '../lib/types/index';
import { ChatUser, FormattedMessage, MessageMedia, eventClassNames } from '../types';
import { getUserInfo } from './client';
import terminalSize from 'term-size';
import terminalImage from 'terminal-image';

export const sendMessage = async (
	client: TelegramClient,
	userToSend: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	message: string,
	isReply?: boolean | undefined,
	replyToMessageId?: number,
) => {
	if (!client.connected) await client.connect();

	const sendMessageParam = {
		message: message,
		...isReply && { replyTo: replyToMessageId }
	}
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
		const result = await client.deleteMessages(new Api.InputPeerUser({
			userId: userToSend?.peerId,
			accessHash: userToSend?.accessHash
		}), [Number(messageId)], { revoke: true })
		return result
	} catch (err) {
		console.log(err)
	}
}

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
		})
		const result = await client.invoke(
			new Api.messages.EditMessage({
				peer: entity,
				id: messageId,
				message: newMessage,
			})
		)
		return result
	} catch (err) {
		console.log(err)
	}
}


export async function getAllMessages(
	client: TelegramClient,
	{ accessHash, firstName, peerId: userId }: ChatUser,
	offsetId?: number
): Promise<FormattedMessage[]> {
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

				const { columns, rows } = terminalSize();
				const maxWidth = Math.floor(columns * 0.8);
				const maxHeight = Math.floor(rows * 0.8);

				return {
					id: message.id,
					sender: message.out ? 'you' : firstName,
					content: message.message,
					isFromMe: !!message.out,
					media: buffer
						? await terminalImage.buffer(new Uint8Array(buffer), {
							width: maxWidth,
							height: maxHeight,
							preserveAspectRatio: true
						})
						: null
				};
			})
		)
	)
		?.map(({ content, ...rest }) => ({ content: content?.trim(), ...rest }))
		?.filter((msg) => msg?.content?.length > 0);

	return orgnizedMessages;
}
export const listenForUserMessages = async (
	client: TelegramClient,
	onMessage: (message: FormattedMessage) => void
) => {
	if (!client.connected) await client.connect();
	client.addEventHandler(async (event) => {
		const userId = event.userId;
		if (userId) {
			const user = (await getUserInfo(client, userId)) as unknown as User;
			const isNewMessage =
				(event.className as (typeof eventClassNames)[number]) === 'UpdateShortMessage';

			if (isNewMessage) {
				onMessage({
					id: event.id,
					sender: event.out ? 'you' : user.firstName,
					content: event.message,
					isFromMe: event.out,
					media: null
				});
			}
		}
	});
};
