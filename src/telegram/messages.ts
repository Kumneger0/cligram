import { Api, TelegramClient } from 'telegram';
import { MessagesSlice, User } from '../lib/types/index';
import { ChatUser, FormattedMessage, eventClassNames } from '../types';
import { getUserChats, getUserInfo } from './client';

export const sendMessage = async (
	client: TelegramClient,
	userToSend: { peerId: bigInt.BigInteger; accessHash: bigInt.BigInteger },
	message: string
) => {
	if (!client.connected) await client.connect();
	client.sendMessage(
		new Api.InputPeerUser({
			userId: userToSend?.peerId,
			accessHash: userToSend?.accessHash
		}),
		{
			message: message
		}
	);
};

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
				// const media = message.media as MessageMedia

				return {
					id: message.id,
					sender: message.out ? 'you' : firstName,
					content: message.message,
					isFromMe: !!message.out,
					//TODO: implement media download later after updatin the layout
					media: null
					// media: media && media.className == 'MessageMediaPhoto' ? await downloadMedia({ media, size: 'large' }) : null,
				};
			})
		)
	)
		?.map(({ content, ...rest }) => ({ content: content?.trim(), ...rest }))
		?.filter((msg) => msg?.content?.length > 0);

	return orgnizedMessages;
}

export async function getConversationHistory(
	client: TelegramClient,
	{ accessHash, firstName, peerId: userId }: ChatUser,
	limit: number = 100
): Promise<FormattedMessage[]> {
	if (!client.connected) await client.connect();

	const result = (await client.invoke(
		new Api.messages.GetHistory({
			peer: new Api.InputPeerUser({
				userId: userId as unknown as bigInt.BigInteger,
				accessHash
			}),
			limit
		})
	)) as unknown as MessagesSlice;

	return await Promise.all(
		result.messages.reverse().map(async (message): Promise<FormattedMessage> => {
			// const media = message.media as MessageMedia

			return {
				id: message.id,
				sender: message.out ? 'you' : firstName,
				content: message.message,
				isFromMe: message.out,
				//TODO: implement media download later after updatin the layout
				media: null
				// media: media && media.className == 'MessageMediaPhoto' ? await downloadMedia({ media, size: 'large' }) : null,
			};
		})
	);
}
export const listenForUserMessages = async (
	client: TelegramClient,
	setUserChats: React.Dispatch<React.SetStateAction<ChatUser[]>>,
	selectedName: string
) => {
	if (!client.connected) await client.connect();
	console.log('Listening for messages');

	client.addEventHandler(async (event) => {
		const userId = event.userId;
		if (userId) {
			const user = (await getUserInfo(client, userId)) as unknown as User;

			const isNewMessage =
				(event.className as (typeof eventClassNames)[number]) === 'UpdateShortMessage';

			if (isNewMessage) {
				if (user.firstName !== selectedName) {
					const users = await getUserChats(client);
					const userChats = users
						.filter(({ isBot }) => !isBot)
						.map(({ firstName, ...rest }) => ({
							firstName: firstName === selectedName ? firstName + ' *' : firstName,
							...rest
						}));
					setUserChats(userChats);
				}
			}
		}
	});
};
